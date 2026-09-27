package viewer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// How the viewer page was put on screen, for the line the command prints.
const (
	openedAppWindow = "a window of its own"
	openedTab       = "a browser tab"
)

// open puts the viewer URL on screen and reports how.
//
// A Chromium-based browser in `--app=` mode gives a window with no address bar,
// tabs or bookmarks, which is what makes watching a session feel like a tool
// rather than a web page. It is worth preferring for that alone, and it costs
// nothing: it uses a browser that is already installed, unlike bundling a
// webview toolkit, which would put GTK and WebView2 headers between a user and
// `go install`.
//
// Anything unexpected falls back to an ordinary tab, because a window that looks
// nice is not worth a command that fails.
func open(url string, appWindow bool) (string, error) {
	if appWindow {
		// `fit` tells the page it may size the window to the session's picture.
		// It is added only on this path, and only once the launch has actually
		// worked, because the permission is untrue anywhere else: a tab lives in
		// a window full of the user's own tabs. The browser cannot be asked for a
		// size up front — --window-size is ignored when an instance is already
		// running, since that instance opens the window.
		if cmd, ok := chromiumAppCommand(url + "&fit=1"); ok {
			if err := cmd.Start(); err == nil {
				// Not waited on: the window outlives the launcher, and when a
				// browser is already running it is that process which opens the
				// window while this one exits at once.
				go func() { _ = cmd.Wait() }()
				return openedAppWindow, nil
			}
		}
	}
	return openedTab, openDefaultBrowser(url)
}

// openDefaultBrowser hands the URL to whatever the desktop uses for http links.
func openDefaultBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		// Not `cmd /c start`: that would reinterpret the URL's & and ^.
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// chromiumAppCommand builds a command that opens url as its own window, using
// the first Chromium-based browser it can find.
func chromiumAppCommand(url string) (*exec.Cmd, bool) {
	args := []string{"--app=" + url}

	switch runtime.GOOS {
	case "darwin":
		// A macOS app is a bundle, so it is launched through `open` rather than
		// by path. -n forces a new instance, -a names the bundle.
		for _, name := range []string{"Google Chrome", "Microsoft Edge", "Chromium", "Brave Browser"} {
			if macAppExists(name) {
				return exec.Command("open", append([]string{"-na", name, "--args"}, args...)...), true
			}
		}
	case "windows":
		// No standard install location, so look in the three places installers
		// actually use, per-machine and per-user.
		roots := []string{
			os.Getenv("ProgramFiles"),
			os.Getenv("ProgramFiles(x86)"),
			os.Getenv("LocalAppData"),
		}
		suffixes := []string{
			`Google\Chrome\Application\chrome.exe`,
			`Microsoft\Edge\Application\msedge.exe`,
			`Chromium\Application\chrome.exe`,
			`BraveSoftware\Brave-Browser\Application\brave.exe`,
		}
		for _, suffix := range suffixes {
			for _, root := range roots {
				if root == "" {
					continue
				}
				path := filepath.Join(root, suffix)
				if isFile(path) {
					return exec.Command(path, args...), true
				}
			}
		}
	default:
		for _, name := range []string{
			"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
			"microsoft-edge", "microsoft-edge-stable", "brave-browser",
		} {
			if path, err := exec.LookPath(name); err == nil {
				return exec.Command(path, args...), true
			}
		}
	}
	return nil, false
}

func macAppExists(name string) bool {
	candidates := []string{filepath.Join("/Applications", name+".app")}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "Applications", name+".app"))
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// describeOpen is the line `view` prints about where the page went, which is
// worth saying because an app window can appear behind whatever has focus.
func describeOpen(mode string) string {
	return fmt.Sprintf("Opened in %s.", mode)
}
