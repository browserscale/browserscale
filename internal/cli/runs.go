package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	browserscale "github.com/browserscale/browserscale-go"
)

// The `runs` group covers the scripts inside a session, as opposed to the
// sessions themselves.
//
// It is a group rather than flags on the top-level commands because of one
// collision: stopping a run and stopping a session are both destructive, and
// they destroy different things. `browserscale stop <id>` throws away a browser;
// `browserscale runs stop <id>` cancels a script and leaves the browser alone.
// Had the second been a flag on the first, a typo would decide between them.

func runRuns(args []string) error {
	if len(args) == 0 {
		printRunsUsage()
		return errUsage
	}
	switch args[0] {
	case "list":
		return runRunsList(args[1:])
	case "stop":
		return runRunsStop(args[1:])
	case "follow":
		return runRunsFollow(args[1:])
	case "help", "--help", "-h":
		printRunsUsage()
		return nil
	default:
		return fmt.Errorf("unknown runs command %q (run `browserscale runs help`)", args[0])
	}
}

func printRunsUsage() {
	fmt.Fprint(os.Stderr, `Usage: browserscale runs <command> [flags]

Commands:
  list    <session-id>            Scripts currently running in a session
  stop    <session-id> [run-id]   Cancel a script (or all of them), keeping the browser
  follow  <session-id> [run-id]   Stream a running script's console output

A run is a script inside a session. Stopping a run leaves the browser exactly as
the script left it — use "browserscale stop" to end the session itself.
`)
}

func runRunsList(args []string) error {
	f := newFleetFlags("runs list", `Usage: browserscale runs list <session-id>

Lists the scripts still executing in a session.

Only running scripts appear. A finished run is reported once to whoever was
watching and then forgotten, so an empty list means nothing is running — not that
nothing ever ran.
`)
	if err := f.parse(args); err != nil {
		return err
	}
	rest := f.args()
	if len(rest) != 1 {
		return usageError(f.flagSet, "runs list needs one session id")
	}

	ctx := context.Background()
	browser, err := connect(ctx, f.resolved, rest[0])
	if err != nil {
		return err
	}
	defer browser.CloseConn()

	runs, err := browser.ListScriptRuns(ctx)
	if err != nil {
		return err
	}

	if *f.asJSON {
		return printJSON(runs)
	}
	if len(runs) == 0 {
		fmt.Println("No scripts running in this session.")
		return nil
	}
	w := newTable()
	fmt.Fprintln(w, "RUN\tRUNNING")
	for _, run := range runs {
		fmt.Fprintf(w, "%s\t%s\n", run.RunId, shortDuration(run.Running))
	}
	return w.Flush()
}

func runRunsStop(args []string) error {
	f := newFleetFlags("runs stop", `Usage: browserscale runs stop <session-id> [run-id]

Cancels a script running in a session. Without a run id, cancels every script in
that session.

The browser is left running and untouched — whatever the script had already done
to the page stays done. A script mid-way through a checkout will not be undone.
`)
	if err := f.parse(args); err != nil {
		return err
	}
	rest := f.args()
	if len(rest) == 0 || len(rest) > 2 {
		return usageError(f.flagSet, "runs stop needs a session id and optionally a run id")
	}
	runID := ""
	if len(rest) == 2 {
		runID = rest[1]
	}

	ctx := context.Background()
	browser, err := connect(ctx, f.resolved, rest[0])
	if err != nil {
		return err
	}
	defer browser.CloseConn()

	stopped, err := browser.StopScripts(ctx, runID)
	if err != nil {
		return err
	}
	if *f.asJSON {
		return printJSON(map[string]any{"stopped": stopped})
	}
	if stopped == 0 {
		if runID != "" {
			fmt.Printf("No run %s in flight.\n", runID)
		} else {
			fmt.Println("Nothing was running.")
		}
		return nil
	}
	fmt.Printf("Cancelled %d %s\n", stopped, plural(stopped, "run", "runs"))
	return nil
}

func runRunsFollow(args []string) error {
	f := newFleetFlags("runs follow", `Usage: browserscale runs follow <session-id> [run-id]

Streams the console output of scripts already running in a session. Without a run
id, follows every script in it.

Only output produced from now on arrives — a line printed before this command
started is not kept anywhere. Following does not affect the run: detaching with
Ctrl-C leaves the script going.
`)
	if err := f.parse(args); err != nil {
		return err
	}
	rest := f.args()
	if len(rest) == 0 || len(rest) > 2 {
		return usageError(f.flagSet, "runs follow needs a session id and optionally a run id")
	}
	runID := ""
	if len(rest) == 2 {
		runID = rest[1]
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()

	browser, err := connect(ctx, f.resolved, rest[0])
	if err != nil {
		return err
	}
	defer browser.CloseConn()

	// Tell the user what is actually running before going quiet, so following a
	// session with nothing in it is obvious rather than looking like a hang.
	if runID == "" {
		runs, err := browser.ListScriptRuns(ctx)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			fmt.Fprintln(os.Stderr, "Nothing is running in this session yet — waiting. Ctrl-C to stop.")
		}
	}

	follow, err := browser.FollowScript(ctx, runID, func(event browserscale.ScriptEvent) {
		switch {
		case event.Log != nil:
			// The run id goes on the line when following a whole session, since
			// two scripts' output would otherwise be interleaved with no way to
			// tell them apart. It has to be part of the same write: printed
			// separately it could land on a different line once two runs are
			// producing output.
			prefix := ""
			if runID == "" {
				prefix = fmt.Sprintf("[%s] ", event.RunId)
			}
			printScriptLine(prefix, *event.Log)
		case event.Finished != nil:
			status := "finished"
			if event.Finished.Stopped {
				status = "cancelled"
			} else if !event.Finished.Success {
				status = "failed"
			}
			fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", event.RunId, status, event.Finished.Result)
		}
	})
	if err != nil {
		return err
	}
	defer follow.Stop()

	if err := follow.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}
