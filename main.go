// Command browserscale is the browserscale.cloud developer CLI.
//
// Two halves. `init` and `dev` work on a module in the current directory; `login`,
// `list`, `rent`, `run`, `stop` and `runs` work on the cloud browsers an API key
// holds.
package main

import (
	"fmt"
	"os"

	"github.com/browserscale/browserscale/internal/cli"
)

func main() {
	err := cli.Run(os.Args[1:])
	if err == nil {
		return
	}
	// Some failures have already said everything there is to say: a usage error
	// printed the usage, a failed script printed its own message, an interrupt was
	// the user's own doing. Prefixing those with "error:" would only add noise, so
	// they carry the exit code and nothing else.
	if cli.Silent(err) {
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
