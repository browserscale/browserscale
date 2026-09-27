// Package scaffold renders a new browserscale automation module from a Spec.
package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// refDocsFS holds the browserscale docs as individual Markdown files (guides +
// the Go API reference + an index), mirrored from the docs pipeline by
// generate:llm-docs (CI checks it). The whole tree is written into every
// generated module under docs/ so an agent can open a specific page or grep it
// offline instead of guessing method names.
//
//go:embed docs
var refDocsFS embed.FS

// file maps an embedded template to its output filename in the new module.
type file struct {
	tmpl string
	out  string
}

// baseFiles are rendered for every module regardless of lifecycle kind. Every
// module runs proxied, so the (commented) proxy list template ships for all
// kinds under the runtime Files/ dir the configurator picks from.
var baseFiles = []file{
	{"gomod.tmpl", "go.mod"},
	{"main.tmpl", "main.go"},
	{"rent.tmpl", "rent.go"},
	{"agents.tmpl", "AGENTS.md"},
	{"readme.tmpl", "README.md"},
	{"gitignore.tmpl", ".gitignore"},
	{"manifest.tmpl", "browserscale.yaml"},
	{"proxies_txt.tmpl", "data/proxies.txt"},
}

// moduleTemplateForKind selects the module.go template. The queue kind ships a
// CSV-backed variant (typed records + a configurator content check).
func moduleTemplateForKind(kind string) string {
	if kind == KindQueue {
		return "module_queue.tmpl"
	}
	return "module.tmpl"
}

// flowFilesForKind selects the browser-flow templates. Every kind ships a thin
// flow.go (doTask as dispatcher) plus one named-flow file that holds the
// worked example — register.go for queue/one-shot, task.go otherwise. Agents
// are expected to add further flows as sibling files (enter.go, login.go, …)
// and keep doTask as the switchboard.
func flowFilesForKind(kind string) []file {
	switch kind {
	case KindQueue:
		return []file{
			{"flow_queue.tmpl", "flow.go"},
			{"register_queue.tmpl", "register.go"},
		}
	case KindOneShot:
		return []file{
			{"flow_oneshot.tmpl", "flow.go"},
			{"register_oneshot.tmpl", "register.go"},
		}
	default:
		return []file{
			{"flow.tmpl", "flow.go"},
			{"task.tmpl", "task.go"},
		}
	}
}

// runTemplateForKind selects the run.go template matching the lifecycle.
func runTemplateForKind(kind string) string {
	switch kind {
	case KindOneShot:
		return "run_oneshot.tmpl"
	case KindQueue:
		return "run_queue.tmpl"
	case KindRepeat:
		return "run_repeat.tmpl"
	case KindContinuous:
		return "run_continuous.tmpl"
	case KindScheduled:
		return "run_scheduled.tmpl"
	default:
		return "run_custom.tmpl"
	}
}

// Generate writes the module described by spec into dir. dir must not
// already contain files.
func Generate(dir string, spec Spec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	if err := ensureEmptyDir(dir); err != nil {
		return err
	}

	files := append([]file{}, baseFiles...)
	files = append(files,
		file{moduleTemplateForKind(spec.Kind), "module.go"},
		file{runTemplateForKind(spec.Kind), "run.go"},
	)
	files = append(files, flowFilesForKind(spec.Kind)...)
	if spec.Kind == KindQueue {
		// The queue kind additionally ships a ready-to-edit accounts CSV (the
		// proxy list ships for every kind via baseFiles).
		files = append(files, file{"accounts_csv.tmpl", "data/accounts.csv"})
	}

	for _, f := range files {
		if err := renderFile(dir, f, spec); err != nil {
			return fmt.Errorf("%s: %w", f.out, err)
		}
	}

	// The offline docs are embedded content, not templates — copy the tree
	// verbatim into the module's docs/ (running it through text/template would
	// choke on the Markdown braces).
	if err := writeEmbeddedDocs(dir); err != nil {
		return fmt.Errorf("docs: %w", err)
	}
	return nil
}

// writeEmbeddedDocs copies the embedded docs tree into dir, preserving its
// layout (embedded paths are "docs/..." so they land under <dir>/docs/...).
func writeEmbeddedDocs(dir string) error {
	return fs.WalkDir(refDocsFS, "docs", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		content, rerr := refDocsFS.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		return writeRaw(filepath.Join(dir, filepath.FromSlash(p)), content)
	})
}

// writeRaw writes content to outPath verbatim, creating parent dirs.
func writeRaw(outPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, content, 0o644)
}

func renderFile(dir string, f file, spec Spec) error {
	raw, err := templatesFS.ReadFile("templates/" + f.tmpl)
	if err != nil {
		return err
	}

	tmpl, err := template.New(f.tmpl).Parse(string(raw))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return err
	}

	content := buf.Bytes()
	if strings.HasSuffix(f.out, ".go") {
		formatted, ferr := format.Source(content)
		if ferr != nil {
			// Keep the raw output so the failure is inspectable rather than
			// silently swallowed; surface the formatting error.
			_ = os.WriteFile(filepath.Join(dir, f.out), content, 0o644)
			return fmt.Errorf("format: %w", ferr)
		}
		content = formatted
	}

	outPath := filepath.Join(dir, f.out)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, content, 0o644)
}

func ensureEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o755)
		}
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("directory %q is not empty", dir)
	}
	return nil
}
