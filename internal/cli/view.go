package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/browserscale/browserscale/internal/viewer"
)

func runView(args []string) error {
	f := newFleetFlags("view", `Usage: browserscale view <session-id>

Watches a session's screen, and by default takes your mouse and keyboard.

This opens a window of its own, drawn by a browser you already have, rather than
drawing in the terminal. The stream is H.264 over WebRTC, so the thing that can
decode and display it well is already on your desktop. What this command adds is
the half a browser tab cannot do for itself: it holds your API key and signs the
calls, so watching works with nothing but a key — no web login, and against a
local or private deployment just as well as the public one.

The window has no address bar or tabs, which needs a Chromium-based browser
(Chrome, Edge, Brave). Without one it falls back to an ordinary tab, as does
-tab.

Video goes from the engine to your browser directly over the relay; it does not
pass through this process.

Ctrl+. opens DevTools beside the picture: Elements, Network, Console and
Actions, signed by this process like everything else.

For looking at a session without touching it — one an automation is driving,
where a stray click would change the outcome — use -read-only.
`)
	var (
		readOnly = f.flagSet.Bool("read-only", false, "watch without sending mouse or keyboard")
		port     = f.flagSet.Int("port", 0, "port for the local viewer (default: a free one)")
		noOpen   = f.flagSet.Bool("no-open", false, "print the URL instead of opening a browser")
		asTab    = f.flagSet.Bool("tab", false, "open an ordinary browser tab instead of a window")
	)
	if err := f.parse(args); err != nil {
		return err
	}
	rest := f.args()
	if len(rest) == 0 {
		return usageError(f.flagSet, "view needs a session id")
	}
	if len(rest) > 1 {
		return usageError(f.flagSet, "view takes one session id, got %d", len(rest))
	}
	sessionID := rest[0]

	// Ctrl+C is how this command ends — there is no other finish line — so it
	// has to be a clean stop that frees the engine's encoder, not a kill.
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()

	browser, err := connect(ctx, f.resolved, sessionID)
	if err != nil {
		return err
	}
	// CloseConn, not Close: this detaches from the session and leaves the rental
	// running, which is the whole premise of watching one.
	defer browser.CloseConn()

	// The URL is this command's one piece of output, so it goes to stdout and
	// the remarks around it to stderr — `url=$(browserscale view -no-open id)`
	// then does the obvious thing.
	log := io.Writer(os.Stderr)
	ready := func(url string) error {
		_, err := fmt.Println(url)
		return err
	}
	if *f.asJSON {
		log = io.Discard
		ready = func(url string) error {
			return printJSON(map[string]any{"sessionId": sessionID, "url": url})
		}
	}

	err = viewer.Serve(ctx, viewer.Options{
		Browser:     browser,
		SessionID:   sessionID,
		Port:        *port,
		Interactive: !*readOnly,
		Open:        !*noOpen,
		AppWindow:   !*asTab,
		Ready:       ready,
		Log:         log,
	})
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return errInterrupted
	}
	return nil
}
