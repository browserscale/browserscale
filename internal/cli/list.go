package cli

import (
	"context"
	"fmt"
	"time"

	browserscale "github.com/browserscale/browserscale-go"
)

func runList(args []string) error {
	f := newFleetFlags("list", `Usage: browserscale list [flags]

Lists the browser sessions this API key is currently paying for.

Every other fleet command takes a session id, and this is where one comes from:
an id is only otherwise known to the process that rented it, so a crashed script
leaves sessions running that nothing can name.
`)
	if err := f.parse(args); err != nil {
		return err
	}
	if rest := f.args(); len(rest) > 0 {
		return usageError(f.flagSet, "list takes no arguments, got %q", rest[0])
	}

	ctx := context.Background()
	browsers, err := browserscale.ListBrowsers(ctx, f.resolved)
	if err != nil {
		return err
	}

	if *f.asJSON {
		return printJSON(browsers)
	}

	if len(browsers) == 0 {
		fmt.Println("No running sessions. `browserscale rent` starts one.")
		return nil
	}

	now := time.Now()
	w := newTable()
	fmt.Fprintln(w, "SESSION\tCOUNTRY\tAGE\tREMAINING\tEGRESS\tPROXY")
	for _, b := range browsers {
		remaining := "unlimited"
		if b.RemainingSeconds != nil {
			remaining = shortDuration(time.Duration(*b.RemainingSeconds) * time.Second)
		}
		age := shortDuration(now.Sub(time.Unix(b.StartTime, 0)))
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			b.SessionId,
			orDash(b.CountryCode),
			age,
			remaining,
			orDash(b.PublicIp),
			orDash(b.ProxyHost),
		)
	}
	return w.Flush()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
