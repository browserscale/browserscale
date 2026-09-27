package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateQueueRentAndStatus(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{
		Name:      "demo",
		GoModule:  "demo",
		Kind:      KindQueue,
		Threading: ThreadSingle,
	}
	if err := Generate(dir, spec); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	rent, err := os.ReadFile(filepath.Join(dir, "rent.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(rent)
	if strings.Contains(body, "WithCountryCode(\"US\")") {
		t.Error("rent.go must not suggest WithCountryCode as the default example")
	}
	if !strings.Contains(body, "auto-match") && !strings.Contains(body, "automatically") {
		t.Error("rent.go should document bare-rent proxy geo matching")
	}
	run, err := os.ReadFile(filepath.Join(dir, "run.go"))
	if err != nil {
		t.Fatal(err)
	}
	runBody := string(run)
	for _, want := range []string{"ErrInspect", "ErrRetry", "runStats", "OK:", "Inspect:"} {
		if !strings.Contains(runBody, want) {
			t.Errorf("run.go missing %q", want)
		}
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "WithCountryCode") {
		t.Error("AGENTS.md should warn about WithCountryCode overrides")
	}
	if !strings.Contains(string(agents), "data/config.json") {
		t.Error("AGENTS.md should document data/config.json")
	}
	if !strings.Contains(string(agents), "browserscale dev") {
		t.Error("AGENTS.md should tell agents to use browserscale dev")
	}
	if strings.Contains(string(agents), "dev --") {
		t.Error("AGENTS.md should not teach browserscale dev -- …")
	}
	for _, name := range []string{"data/proxies.txt", "data/accounts.csv"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "Files")); err == nil {
		t.Error("legacy data/Files/ should not be generated")
	}
}

func TestGenerateFlowSplit(t *testing.T) {
	cases := []struct {
		kind      string
		wantExtra string
	}{
		{KindQueue, "register.go"},
		{KindOneShot, "register.go"},
		{KindRepeat, "task.go"},
		{KindContinuous, "task.go"},
		{KindScheduled, "task.go"},
		{KindCustom, "task.go"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			dir := t.TempDir()
			spec := Spec{
				Name:      "demo",
				GoModule:  "demo",
				Kind:      tc.kind,
				Threading: ThreadSingle,
			}
			if err := Generate(dir, spec); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			for _, name := range []string{"flow.go", tc.wantExtra, "run.go", "module.go", "AGENTS.md"} {
				if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
					t.Errorf("missing %s: %v", name, err)
				}
			}
			body, err := os.ReadFile(filepath.Join(dir, "flow.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "return m.") {
				t.Errorf("flow.go should dispatch to a named flow, got:\n%s", body)
			}
			// Named-flow file must not redefine doTask.
			extra, err := os.ReadFile(filepath.Join(dir, tc.wantExtra))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(extra), "func (m *Module) doTask") {
				t.Errorf("%s must not define doTask", tc.wantExtra)
			}
		})
	}
}
