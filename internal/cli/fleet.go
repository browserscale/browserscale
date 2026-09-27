package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	browserscale "github.com/browserscale/browserscale-go"
)

// Shared plumbing for the commands that act on rented browsers, as opposed to
// the ones (init, dev) that act on a directory.
//
// The split is worth keeping visible: everything here needs an API key, reaches
// a machine somewhere else, and can cost money or destroy work in progress. The
// local commands need none of that and can be retried freely.

// fleetFlags are the flags every fleet command accepts.
type fleetFlags struct {
	key      *string
	asJSON   *bool
	flagSet  *flag.FlagSet
	resolved string
}

// newFleetFlags registers -key and -json on a fresh flag set.
func newFleetFlags(name, usage string) *fleetFlags {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	f := &fleetFlags{
		key:     fs.String("key", "", "API key (default: login, BROWSERSCALE_API_KEY, or the module's config)"),
		asJSON:  fs.Bool("json", false, "print machine-readable JSON instead of a table"),
		flagSet: fs,
	}
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		fmt.Fprint(os.Stderr, "\nFlags:\n")
		fs.PrintDefaults()
	}
	return f
}

// parse handles the flags and resolves the API key, so a command body can
// assume both are settled.
func (f *fleetFlags) parse(args []string) error {
	if err := f.flagSet.Parse(reorderFlagsFirst(f.flagSet, args)); err != nil {
		return err
	}
	applyEndpoint()
	key, source, err := resolveAPIKey(*f.key)
	if err != nil {
		return err
	}
	f.resolved = key
	// Worth saying only when it could surprise: a key picked up from the
	// directory you happen to be standing in is not obvious, and it decides
	// which account `stop` acts on.
	if source == keyFromModule && !*f.asJSON {
		fmt.Fprintf(os.Stderr, "Using the API key from %s\n", source)
	}
	return nil
}

// args returns the positional arguments left after the flags.
func (f *fleetFlags) args() []string { return f.flagSet.Args() }

// findBrowser looks up one session by id.
//
// It goes through the listing rather than taking the id on trust, for two
// reasons. The gRPC URL a session is driven from is not derivable from its id,
// so something has to fetch it. And a wrong id is the most likely mistake when
// ids are copied by hand, which is worth a clear answer instead of a connection
// that fails later for reasons that read like an outage.
func findBrowser(ctx context.Context, apiKey, sessionID string) (browserscale.BrowserInfo, error) {
	browsers, err := browserscale.ListBrowsers(ctx, apiKey)
	if err != nil {
		return browserscale.BrowserInfo{}, err
	}
	for _, b := range browsers {
		if b.SessionId == sessionID {
			return b, nil
		}
	}
	if len(browsers) == 0 {
		return browserscale.BrowserInfo{}, fmt.Errorf("no session %s — this key holds no running sessions", sessionID)
	}
	ids := make([]string, 0, len(browsers))
	for _, b := range browsers {
		ids = append(ids, b.SessionId)
	}
	return browserscale.BrowserInfo{}, fmt.Errorf("no session %s. Running: %s", sessionID, strings.Join(ids, ", "))
}

// connect attaches to a running session, returning a handle whose CloseConn
// detaches without ending the rental.
func connect(ctx context.Context, apiKey, sessionID string) (*browserscale.CloudBrowser, error) {
	info, err := findBrowser(ctx, apiKey, sessionID)
	if err != nil {
		return nil, err
	}
	return info.Connect(ctx, apiKey)
}

// printJSON writes one value as indented JSON, which is what every -json output
// goes through so the shape is consistent.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newTable returns a tabwriter set up the way every table here prints: columns
// separated by two spaces, no padding characters, so the output stays greppable.
func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// shortDuration formats a span the way a human scanning a list reads it: the two
// largest units, and never more precision than the number deserves.
func shortDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// parseDuration accepts both a bare number of seconds and a Go duration, because
// both readings of "-duration 600" are reasonable and guessing wrong costs a
// rental. A bare number is seconds, matching the API; anything with a unit is
// parsed as written.
func parseDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d, nil
	}
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err != nil {
		return 0, fmt.Errorf("cannot read %q as a duration — use seconds (600) or a unit (10m)", value)
	}
	if !isAllDigits(value) {
		return 0, fmt.Errorf("cannot read %q as a duration — use seconds (600) or a unit (10m)", value)
	}
	return time.Duration(seconds) * time.Second, nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// reorderFlagsFirst moves flags ahead of positional arguments.
//
// The flag package stops parsing at the first argument that is not a flag, so
// `run abc script.js -json` would silently ignore -json — the command would run
// and print the wrong format, with nothing to indicate why. Since every fleet
// command takes an id first, that is the order people naturally type. Reordering
// here makes the two forms equivalent instead of one of them quietly wrong.
//
// Whether a flag consumes the next argument is read off the flag set rather than
// guessed, so `-key sk_x <id>` keeps its value. A lone "-" is left alone: `run`
// uses it to mean stdin. Everything after "--" is positional by definition.
func reorderFlagsFirst(fs *flag.FlagSet, args []string) []string {
	flags := make([]string, 0, len(args))
	positional := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		if strings.Contains(arg, "=") {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		// An unknown flag is left as a lone token: Parse will reject it, which is
		// a better answer than swallowing the next argument on a guess.
		if def := fs.Lookup(name); def != nil && !isBoolFlag(def.Value) && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return append(flags, positional...)
}

// isBoolFlag reports whether a flag is the kind that stands alone, using the
// same interface the flag package itself checks for.
func isBoolFlag(v flag.Value) bool {
	bf, ok := v.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}

// errUsage is returned when a command was called with the wrong positional
// arguments, so main can tell that apart from a failure out in the world.
var errUsage = errors.New("usage")

func usageError(fs *flag.FlagSet, format string, args ...any) error {
	fmt.Fprintf(os.Stderr, format+"\n\n", args...)
	fs.Usage()
	return errUsage
}
