package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// runDev builds the module in dir, kills any previous `browserscale dev`
// child for that dir, starts it headless (-yes), and streams its logs until
// Ctrl+C. Re-running the same command rebuilds and restarts.
//
// Always non-interactive: requires data/config.json. Without -run the harness
// picks a timestamp folder under runs/; with -run NAME it uses that folder.
func runDev(args []string) error {
	dir := "."
	runName := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			printDevUsage()
			return nil
		case a == "-dir" || a == "--dir":
			if i+1 >= len(args) {
				return errors.New("-dir requires a path")
			}
			i++
			dir = args[i]
		case a == "-run" || a == "--run":
			if i+1 >= len(args) {
				return errors.New("-run requires a name")
			}
			i++
			runName = args[i]
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag %q (run `browserscale dev -h`)", a)
		default:
			return fmt.Errorf("unexpected argument %q (run `browserscale dev -h`)", a)
		}
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		return fmt.Errorf("%s: no go.mod — run from a module directory or pass -dir", abs)
	}
	configPath := filepath.Join(abs, "data", "config.json")
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("%s missing — run once with `go run .` to create it via the configurator", configPath)
	}

	goBin, err := lookGo()
	if err != nil {
		return err
	}

	stateDir := filepath.Join(abs, ".browserscale")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	binName := "module"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(stateDir, binName)
	pidPath := filepath.Join(stateDir, "dev.pid")

	if err := stopPrevious(pidPath); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not stop previous process:", err)
	}

	fmt.Fprintln(os.Stderr, "browserscale dev: building…")
	build := exec.Command(goBin, "build", "-o", binPath, ".")
	build.Dir = abs
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Always headless. -run is optional (timestamp when omitted).
	modArgs := []string{"-yes"}
	if runName != "" {
		modArgs = append(modArgs, "-run", runName)
	}

	cmd := exec.Command(binPath, modArgs...)
	cmd.Dir = abs
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not write pid file:", err)
	}
	if runName != "" {
		fmt.Fprintf(os.Stderr, "browserscale dev: running pid %d → runs/%s (Ctrl+C to stop; re-run to rebuild+restart)\n", cmd.Process.Pid, runName)
	} else {
		fmt.Fprintf(os.Stderr, "browserscale dev: running pid %d → runs/<timestamp> (Ctrl+C to stop; re-run to rebuild+restart)\n", cmd.Process.Pid)
	}

	go streamLines(stdout, os.Stdout)
	go streamLines(stderr, os.Stderr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	select {
	case sig := <-sigCh:
		fmt.Fprintln(os.Stderr, "browserscale dev: caught", sig, "— stopping")
		_ = killProcess(cmd.Process)
		select {
		case <-waitCh:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-waitCh
		}
		_ = os.Remove(pidPath)
		return nil
	case err := <-waitCh:
		_ = os.Remove(pidPath)
		if err == nil {
			return nil
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("module exited with status %d", ee.ExitCode())
		}
		return err
	}
}

func printDevUsage() {
	fmt.Print(`browserscale dev — build, restart, and stream a module (always headless)

Usage:
  browserscale dev [-dir <path>] [-run <name>]

Requires data/config.json (create it once with go run .). Then:
  1. Checks that 'go' is on PATH
  2. Stops the previous 'browserscale dev' child for that directory
  3. Runs 'go build' into .browserscale/
  4. Starts the binary with -yes (no TUI) and streams stdout/stderr

Without -run the harness creates runs/<timestamp>/. With -run NAME it uses
runs/NAME/ (append OK). Re-run after edits to rebuild and restart. Ctrl+C stops
the child. Cursor/Codex see the same stream in the terminal.
`)
}

func lookGo() (string, error) {
	goBin, err := exec.LookPath("go")
	if err == nil {
		return goBin, nil
	}
	return "", errors.New(`go not found on PATH.

Install Go 1.25+ from https://go.dev/dl/ and ensure the installer added the
Go bin directory to your PATH (open a new terminal after installing).
On Windows, typical location: %USERPROFILE%\go\bin or C:\Program Files\Go\bin`)
}

func stopPrevious(pidPath string) error {
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		_ = os.Remove(pidPath)
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidPath)
		return nil
	}
	if err := killProcess(proc); err != nil {
		// Already gone is fine.
		if !errors.Is(err, os.ErrProcessDone) && !isProcessGone(err) {
			return fmt.Errorf("pid %d: %w", pid, err)
		}
	}
	// Give the OS a moment to release the binary on Windows.
	time.Sleep(200 * time.Millisecond)
	_ = os.Remove(pidPath)
	return nil
}

func killProcess(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		// Process.Kill is TerminateProcess on Windows; good enough for our child.
		return proc.Kill()
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return proc.Kill()
	}
	return nil
}

func isProcessGone(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "process already finished") ||
		strings.Contains(msg, "no such process") ||
		strings.Contains(msg, "the operation completed successfully")
}

func streamLines(r io.Reader, w io.Writer) {
	sc := bufio.NewScanner(r)
	// Long log lines (observations) shouldn't truncate mid-stream.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		fmt.Fprintln(w, sc.Text())
	}
}
