package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	browserscale "github.com/browserscale/browserscale-go"
)

func runRun(args []string) error {
	f := newFleetFlags("run", `Usage: browserscale run <session-id> <script.js>
       browserscale run <session-id> -e "await browser.navigate('https://example.com')"
       browserscale run <session-id> -          (read the script from stdin)

Runs a JavaScript file inside a rented browser and waits for it to finish.

The script does not run in the page. It runs beside the browser in an isolate of
its own and reaches the document through the engine: a cross-origin iframe is
read as plain `+"`contentDocument`"+` with no frame ids anywhere, values come back
as live objects you can assign to rather than snapshots, an element can be
handed straight to `+"`browser.click`"+`, and the page sees nothing injected. Steps
cost microseconds instead of network round trips, so loops are affordable.

A guide for it is still to come.

The log streams to stdout as the script produces it, rather than arriving in one
block when it ends — a script that takes a minute is something you can watch. The
return value follows it, on the same stream. Only the CLI's own remarks go to
stderr.

Use -json when something has to parse the result: that gives one document with the
value and the log as separate fields, instead of a stream meant for a human.

Exits non-zero if the script throws. Ctrl-C cancels the run in the browser rather
than leaving it going.
`)
	var (
		detach = f.flagSet.Bool("detach", false, "start the script and exit, leaving it running")
		expr   = f.flagSet.String("e", "", "run this source instead of a file")
	)
	if err := f.parse(args); err != nil {
		return err
	}

	rest := f.args()
	if len(rest) == 0 {
		return usageError(f.flagSet, "run needs a session id (`browserscale list` shows them)")
	}
	sessionID := rest[0]
	rest = rest[1:]

	source, err := readSource(*expr, rest, f.flagSet)
	if err != nil {
		return err
	}

	// Interrupting is expected — a script that turns out to be wrong gets Ctrl-C
	// — and it has to cancel the run, not just this process. Otherwise the script
	// keeps driving a browser nobody is watching, on a session still being paid
	// for.
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()

	browser, err := connect(ctx, f.resolved, sessionID)
	if err != nil {
		return err
	}
	defer browser.CloseConn()

	if *detach {
		return startDetached(ctx, browser, source, *f.asJSON)
	}
	return runAttached(ctx, browser, source, *f.asJSON)
}

// readSource resolves the script to run from -e, a file, or stdin.
func readSource(expr string, rest []string, fs *flag.FlagSet) (string, error) {
	if expr != "" {
		if len(rest) > 0 {
			return "", usageError(fs, "-e and a script file are two ways to say the same thing — pass one (got %q)", rest[0])
		}
		return expr, nil
	}
	if len(rest) == 0 {
		return "", usageError(fs, "run needs a script file, -e, or - to read stdin")
	}
	if len(rest) > 1 {
		return "", usageError(fs, "run takes one script file, got %d", len(rest))
	}
	if rest[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading the script from stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(rest[0])
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// startDetached launches the script and leaves it running.
func startDetached(ctx context.Context, browser *browserscale.CloudBrowser, source string, asJSON bool) error {
	// The handler is required but nothing reads it here: the events this process
	// would receive are exactly the ones it is walking away from.
	run, err := browser.StartScript(ctx, source, func(browserscale.ScriptEvent) {})
	if err != nil {
		return err
	}
	run.Detach()

	if asJSON {
		return printJSON(map[string]any{"runId": run.RunId(), "detached": true})
	}
	fmt.Println(run.RunId())
	fmt.Fprintf(os.Stderr, "\nRunning detached. Watch it with:\n  browserscale runs follow %s %s\nCancel it with:\n  browserscale runs stop %s %s\n",
		browser.SessionId(), run.RunId(), browser.SessionId(), run.RunId())
	return nil
}

// runAttached launches the script and reports its output until it ends.
func runAttached(ctx context.Context, browser *browserscale.CloudBrowser, source string, asJSON bool) error {
	// In JSON mode the log is part of one document, so it is collected rather
	// than streamed; the handler runs on the SDK's goroutine, hence the lock.
	var (
		mu      sync.Mutex
		entries []logLine
	)

	run, err := browser.StartScript(ctx, source, func(event browserscale.ScriptEvent) {
		if event.Log == nil {
			return
		}
		if asJSON {
			mu.Lock()
			entries = append(entries, logLine{
				Level:     event.Log.Level,
				Message:   event.Log.Message,
				Timestamp: event.Log.Timestamp,
			})
			mu.Unlock()
			return
		}
		printScriptLine("", *event.Log)
	})
	if err != nil {
		return err
	}

	outcome, waitErr := run.Wait(ctx)
	if waitErr != nil {
		if errors.Is(waitErr, context.Canceled) {
			// The signal cancelled our context, so stopping needs a fresh one.
			fmt.Fprintln(os.Stderr, "\nInterrupted — cancelling the run.")
			if err := run.Stop(context.Background()); err != nil {
				return fmt.Errorf("the run could not be cancelled: %w", err)
			}
			return errInterrupted
		}
		return waitErr
	}

	mu.Lock()
	log := entries
	mu.Unlock()

	if asJSON {
		if err := printJSON(map[string]any{
			"runId":   run.RunId(),
			"success": outcome.Success,
			"result":  outcome.Result,
			"stopped": outcome.Stopped,
			"dropped": run.Dropped(),
			"log":     log,
		}); err != nil {
			return err
		}
	} else {
		if dropped := run.Dropped(); dropped > 0 {
			fmt.Fprintf(os.Stderr, "warning: %d log lines were dropped because this reader fell behind\n", dropped)
		}
		if outcome.Result != "" && outcome.Result != "undefined" {
			fmt.Println(outcome.Result)
		}
	}

	if !outcome.Success {
		// The message is already in Result, and printing it again as an error
		// would say it twice. In JSON mode it is in the document.
		if !asJSON {
			return fmt.Errorf("the script failed: %s", outcome.Result)
		}
		return errScriptFailed
	}
	if outcome.Stopped {
		return errors.New("the run was cancelled before it finished")
	}
	return nil
}

type logLine struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// printScriptLine writes one console line as the script produces it.
//
// One rule decides the stream throughout: what the *script* emits goes to stdout,
// what the *CLI* says about it goes to stderr. So a log line is stdout even
// though it is diagnostic in nature — it came from the script — while "N lines
// were dropped" is stderr. Anything else means a reader has to know which of the
// two wrote a given line to make sense of a redirect.
//
// Ordering with the return value is not at risk despite the two writers: Wait
// only returns once the handler has processed the run's final event, so every log
// write happens before the result write.
//
// The level is tagged only when it is not an ordinary log — a prefix on every
// line would be noise on the common case.
func printScriptLine(prefix string, entry browserscale.ScriptLogEntry) {
	switch strings.ToLower(entry.Level) {
	case "", "info", "log":
		fmt.Printf("%s%s\n", prefix, entry.Message)
	default:
		fmt.Printf("%s%s: %s\n", prefix, entry.Level, entry.Message)
	}
}

// errScriptFailed marks a script that threw, for an exit code without a second
// copy of the message on stderr.
var errScriptFailed = errors.New("script failed")

// errInterrupted marks a run the user cancelled, which is not a failure to
// explain — they know.
var errInterrupted = errors.New("interrupted")
