package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	browserscale "github.com/browserscale/browserscale-go"
)

func runRent(args []string) error {
	f := newFleetFlags("rent", `Usage: browserscale rent [flags]

Rents a browser session and prints its id, then leaves it running.

For trying something out by hand: rent one, drive it with `+"`run`"+`, stop it when
done. Production code rents through the SDK instead, so the session's lifetime is
tied to the program that needs it.

The session keeps running after this command exits and is billed until it expires
or is stopped. With no -duration it has no expiry, which means nothing will clean
it up for you.
`)
	var (
		duration = f.flagSet.String("duration", "", "rental length, e.g. 10m or 600 (default: unlimited)")
		country  = f.flagSet.String("country", "", "geo-IP country code, e.g. us")
		timezone = f.flagSet.String("timezone", "", "IANA timezone, e.g. Europe/Berlin")
		proxy    = f.flagSet.String("proxy", "", "proxy as host:port or host:port:user:pass")
	)
	if err := f.parse(args); err != nil {
		return err
	}
	if rest := f.args(); len(rest) > 0 {
		return usageError(f.flagSet, "rent takes no arguments, got %q", rest[0])
	}

	seconds := 0
	if *duration != "" {
		d, err := parseDuration(*duration)
		if err != nil {
			return err
		}
		seconds = int(d.Seconds())
		if seconds <= 0 {
			return fmt.Errorf("-duration must be positive, got %q", *duration)
		}
	}

	host, port, user, pass, err := parseProxy(*proxy)
	if err != nil {
		return err
	}

	cfg := browserscale.NewBrowserConfig(f.resolved, seconds, host, port, user, pass)
	if *country != "" {
		cfg = cfg.WithCountryCode(*country)
	}
	if *timezone != "" {
		cfg = cfg.WithTimezone(*timezone)
	}

	ctx := context.Background()
	browser, err := browserscale.RentBrowser(ctx, cfg)
	if err != nil {
		return err
	}
	// Detach rather than Close: Close would stop the session we just rented.
	// This command's whole output is a session someone else will drive.
	defer browser.CloseConn()

	if *f.asJSON {
		return printJSON(map[string]any{
			"sessionId":   browser.SessionId(),
			"grpcUrl":     browser.GrpcUrl(),
			"countryCode": browser.CountryCode(),
			"timezone":    browser.Timezone(),
			"fingerprint": browser.Fingerprint(),
		})
	}

	// The id alone on stdout, the guidance on stderr, so `id=$(browserscale
	// rent)` does the obvious thing.
	fmt.Println(browser.SessionId())
	fmt.Fprintf(os.Stderr, "\nRunning. Drive it with:\n  browserscale run %s script.js\nStop it with:\n  browserscale stop %s\n",
		browser.SessionId(), browser.SessionId())
	return nil
}

// parseProxy reads host:port or host:port:user:pass. Credentials are positional
// because that is the form proxy vendors hand out.
func parseProxy(value string) (host string, port int, user, pass string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, "", "", nil
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 4 {
		return "", 0, "", "", fmt.Errorf("cannot read %q as a proxy — use host:port or host:port:user:pass", value)
	}
	port, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, "", "", fmt.Errorf("proxy port %q is not a number", parts[1])
	}
	if len(parts) == 4 {
		user, pass = parts[2], parts[3]
	}
	return parts[0], port, user, pass, nil
}
