package cli

import (
	"context"
	"fmt"

	browserscale "github.com/browserscale/browserscale-go"
)

func runStop(args []string) error {
	f := newFleetFlags("stop", `Usage: browserscale stop <session-id>
       browserscale stop -all

Ends a browser session and stops billing for it. Unused credits are refunded.

This discards the session's state — cookies, logins, whatever a script was part
way through. There is no undo and no reconnecting afterwards.

Note that this stops a *session*. To cancel a script running inside one, leaving
the browser alone, use `+"`browserscale runs stop`"+`.
`)
	all := f.flagSet.Bool("all", false, "stop every session this key holds")
	if err := f.parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	rest := f.args()

	if *all {
		if len(rest) > 0 {
			return usageError(f.flagSet, "-all stops everything, so it takes no session id (got %q)", rest[0])
		}
		stopped, err := browserscale.StopAllBrowsers(ctx, f.resolved)
		if err != nil {
			return err
		}
		if *f.asJSON {
			return printJSON(map[string]any{"stopped": stopped})
		}
		fmt.Printf("Stopped %d %s\n", stopped, plural(stopped, "session", "sessions"))
		return nil
	}

	if len(rest) == 0 {
		return usageError(f.flagSet, "stop needs a session id, or -all")
	}
	if len(rest) > 1 {
		return usageError(f.flagSet, "stop takes one session id, got %d", len(rest))
	}

	sessionID := rest[0]
	if _, err := browserscale.StopBrowser(ctx, f.resolved, sessionID); err != nil {
		return err
	}
	if *f.asJSON {
		return printJSON(map[string]any{"stopped": 1, "sessionId": sessionID})
	}
	fmt.Printf("Stopped %s\n", sessionID)
	return nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
