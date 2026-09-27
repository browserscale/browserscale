package scaffold

import "fmt"

// Lifecycle kinds — decide the shape of the generated run.go.
const (
	KindOneShot    = "one-shot"   // run doTask once, then exit
	KindQueue      = "queue"      // drain an input list across workers
	KindRepeat     = "repeat"     // run until N successes, then stop
	KindContinuous = "continuous" // workers loop forever until stopped
	KindScheduled  = "scheduled"  // run on a fixed interval
	KindCustom     = "custom"     // minimal Run, the author writes it
)

// Threading modes.
const (
	ThreadSingle       = "single"       // always one worker
	ThreadFixed        = "fixed"        // a compile-time fixed count
	ThreadConfigurable = "configurable" // -threads flag at runtime
)

// Spec captures the structural answers from the init wizard. It drives
// template rendering and is written out verbatim as browserscale.yaml.
//
// Deliberately structural only — task specifics (proxies, captcha, output,
// region, …) are left for the module author / code agent to add later.
type Spec struct {
	Name         string // module name / target directory
	GoModule     string // go module path (defaults to Name)
	Kind         string
	Threading    string
	FixedThreads int
	CLIVersion   string
}

// Validate checks enum fields and fills sensible fallbacks.
func (s *Spec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("module name is required")
	}
	if s.GoModule == "" {
		s.GoModule = s.Name
	}
	if !oneOf(s.Kind, KindOneShot, KindQueue, KindRepeat, KindContinuous, KindScheduled, KindCustom) {
		return fmt.Errorf("invalid kind %q", s.Kind)
	}
	if !oneOf(s.Threading, ThreadSingle, ThreadFixed, ThreadConfigurable) {
		return fmt.Errorf("invalid threading %q", s.Threading)
	}
	if s.Threading == ThreadFixed && s.FixedThreads < 1 {
		s.FixedThreads = 1
	}
	return nil
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}
