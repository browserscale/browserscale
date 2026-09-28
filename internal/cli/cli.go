// Package cli implements command dispatch for the browserscale CLI.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"runtime/debug"
	"strings"
)

// version is the fallback CLI version, also stamped into scaffolded manifests.
// Release builds overwrite it via
// -ldflags "-X github.com/browserscale/browserscale/internal/cli.version=1.2.3";
// the -dev suffix is what a plain `go build` of a working tree reports.
var version = "0.3.0-dev"

// cliVersion prefers the module version Go records in the binary, because that
// is the one source that cannot go stale: `go install ...@v1.2.3` produces a
// binary whose build info says v1.2.3 no matter what this file claims. Builds
// from a checkout (releases, local `go build`) have no module version, so they
// fall back to the stamped constant above.
func cliVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return version
}

// Run dispatches the given arguments to a subcommand.
//
// The commands fall into two groups that are worth telling apart, because they
// differ in what a mistake costs. `init` and `dev` act on the directory you are
// standing in: no key, no network, and a wrong answer costs a rebuild. The rest
// act on rented browsers: they need an API key, reach a machine elsewhere, and a
// wrong session id can throw away work someone is paying for.
//
// They stay flat in syntax all the same. Grouping them into `browserscale module
// init` and `browserscale browsers list` would make every command longer to pay
// for a distinction the help text already draws.
//
// One rule decides which commands get a noun: the product's own subject is
// implicit, everything else is named. A browser session is what browserscale is
// for, so `list` and `stop` mean sessions. A script run is not, so it lives under
// `runs`.
func Run(args []string) error {
	err := dispatch(args)
	// Asking for help is not a failure. The flag package reports -h as an error
	// so that parsing stops; the usage text has already been printed by then.
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	return err
}

func dispatch(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "dev":
		return runDev(args[1:])
	case "login":
		return runLogin(args[1:])
	case "list":
		return runList(args[1:])
	case "rent":
		return runRent(args[1:])
	case "run":
		return explainScriptAccess(runRun(args[1:]))
	case "view":
		return runView(args[1:])
	case "stop":
		return runStop(args[1:])
	case "runs":
		return explainScriptAccess(runRuns(args[1:]))
	case "version", "--version", "-v":
		fmt.Println("browserscale", cliVersion())
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (run `browserscale help`)", args[0])
	}
}

// Silent reports whether an error has already explained itself, so the caller
// should set an exit code without printing anything further.
func Silent(err error) bool {
	return errors.Is(err, errUsage) ||
		errors.Is(err, errScriptFailed) ||
		errors.Is(err, errInterrupted)
}

func printUsage() {
	fmt.Print(`browserscale ` + cliVersion() + ` — browserscale.cloud developer CLI

Usage:
  browserscale <command> [flags]

Your project:
  init        Scaffold a new automation module
  dev         Build, restart, and stream a module's logs

Your browsers:
  login       Save an API key for the commands below
  list        Show the sessions this key is paying for
  rent        Rent a session and leave it running
  run         Run a script inside a session (BrowserVM, early access)
  view        Watch and drive a session's screen
  stop        End a session
  runs        Work with the scripts inside a session (BrowserVM)

Other:
  version     Print the CLI version
  help        Show this help

Every command takes -h. The browser commands also take -json, and find their API
key from -key, BROWSERSCALE_API_KEY, ` + "`login`" + `, or a module's data/config.json —
in that order.
`)
}
