package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/browserscale/browserscale/internal/prompt"
	"github.com/browserscale/browserscale/internal/scaffold"
)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var (
		name         = fs.String("name", "", "module name (also the target directory)")
		module       = fs.String("module", "", "go module path (default: name)")
		kind         = fs.String("kind", "", "lifecycle: one-shot|queue|repeat|continuous|scheduled|custom")
		threading    = fs.String("threading", "", "threading: single|fixed|configurable")
		fixedThreads = fs.Int("fixed-threads", 4, "worker count when threading=fixed")
		dir          = fs.String("dir", "", "target directory (default: ./<name>)")
		yes          = fs.Bool("yes", false, "accept defaults, skip the interactive wizard")
	)
	fs.BoolVar(yes, "y", false, "alias for -yes")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: browserscale init [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	spec := scaffold.Spec{
		Name:         *name,
		GoModule:     *module,
		Kind:         *kind,
		Threading:    *threading,
		FixedThreads: *fixedThreads,
		CLIVersion:   cliVersion(),
	}

	if !*yes {
		runWizard(&spec)
	} else {
		applyDefaults(&spec)
	}

	if err := spec.Validate(); err != nil {
		return err
	}

	target := *dir
	if target == "" {
		target = filepath.Join(".", spec.Name)
	}

	fmt.Printf("\nScaffolding %q (%s, %s) into %s ...\n", spec.Name, spec.Kind, spec.Threading, target)
	if err := scaffold.Generate(target, spec); err != nil {
		return err
	}

	printNextSteps(target, spec)
	return nil
}

// applyDefaults fills any unset fields with reasonable defaults so that
// `browserscale init -name x -y` works non-interactively (e.g. for agents).
func applyDefaults(s *scaffold.Spec) {
	if s.Kind == "" {
		s.Kind = scaffold.KindOneShot
	}
	if s.Threading == "" {
		s.Threading = scaffold.ThreadConfigurable
	}
}

func runWizard(s *scaffold.Spec) {
	fmt.Println("browserscale init — answer a few questions (enter = default)")
	fmt.Println()

	if s.Name == "" {
		s.Name = prompt.String("Module name", "my-module")
	}
	if s.GoModule == "" {
		s.GoModule = prompt.String("Go module path", s.Name)
	}
	if s.Kind == "" {
		s.Kind = prompt.Select("Lifecycle — how should the module run?", []prompt.Choice{
			{Value: scaffold.KindOneShot, Label: "one-shot — run once, then exit"},
			{Value: scaffold.KindQueue, Label: "queue — drain an input list across workers"},
			{Value: scaffold.KindRepeat, Label: "repeat — run until N successes, then stop"},
			{Value: scaffold.KindContinuous, Label: "continuous — loop forever until stopped"},
			{Value: scaffold.KindScheduled, Label: "scheduled — run on a fixed interval"},
			{Value: scaffold.KindCustom, Label: "custom — minimal Run(), you write the flow"},
		}, scaffold.KindOneShot)
	}
	if s.Threading == "" {
		s.Threading = prompt.Select("Parallelism", []prompt.Choice{
			{Value: scaffold.ThreadSingle, Label: "single — one worker"},
			{Value: scaffold.ThreadConfigurable, Label: "configurable — -threads flag at runtime"},
			{Value: scaffold.ThreadFixed, Label: "fixed — a set worker count"},
		}, scaffold.ThreadConfigurable)
	}
	if s.Threading == scaffold.ThreadFixed {
		s.FixedThreads = prompt.Int("How many workers?", s.FixedThreads)
	}
}

func printNextSteps(dir string, s scaffold.Spec) {
	fmt.Printf(`
Done. Created %s

Next steps:
  cd %s
  go mod tidy
  go run .           # once: configurator → data/config.json
  browserscale dev   # iterate: rebuild → restart → stream (needs config.json)

Where to work:
  flow.go     -> thin dispatcher (doTask picks which named flow to run)
  %s
  module.go   -> config schema (Schema) + module identity
  run.go      -> the %s lifecycle (edit freely, it's your code)
  AGENTS.md   -> browserscale-go SDK + kit reference for your code agent
`, dir, dir, flowHint(s.Kind), s.Kind)
}

func flowHint(kind string) string {
	switch kind {
	case scaffold.KindQueue, scaffold.KindOneShot:
		return "register.go -> YOUR browser flow (worked playground signup); add enter.go / login.go as siblings"
	default:
		return "task.go     -> YOUR browser flow (worked playground example); split into register.go / enter.go as it grows"
	}
}
