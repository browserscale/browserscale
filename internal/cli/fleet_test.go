package cli

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestFlags mirrors what a fleet command registers, so the reordering is
// tested against a real flag set rather than a stand-in.
func newTestFlags() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("key", "", "")
	fs.Bool("json", false, "")
	fs.Bool("all", false, "")
	fs.String("e", "", "")
	return fs
}

func TestReorderFlagsFirst(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "already in order is untouched",
			in:   []string{"-json", "abc"},
			want: []string{"-json", "abc"},
		},
		{
			name: "bool flag after a positional moves ahead of it",
			in:   []string{"abc", "-json"},
			want: []string{"-json", "abc"},
		},
		{
			name: "a flag's value travels with it",
			in:   []string{"abc", "-key", "sk_x"},
			want: []string{"-key", "sk_x", "abc"},
		},
		{
			name: "equals form needs no lookahead",
			in:   []string{"abc", "-key=sk_x", "script.js"},
			want: []string{"-key=sk_x", "abc", "script.js"},
		},
		{
			name: "positional order is preserved",
			in:   []string{"abc", "script.js", "-json"},
			want: []string{"-json", "abc", "script.js"},
		},
		{
			name: "a lone dash is a positional, not a flag",
			in:   []string{"abc", "-", "-json"},
			want: []string{"-json", "abc", "-"},
		},
		{
			name: "everything after -- is positional",
			in:   []string{"-json", "--", "-key", "abc"},
			want: []string{"-json", "-key", "abc"},
		},
		{
			name: "an unknown flag does not swallow the next argument",
			in:   []string{"abc", "-nope", "script.js"},
			want: []string{"-nope", "abc", "script.js"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reorderFlagsFirst(newTestFlags(), tc.in)
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Errorf("reorderFlagsFirst(%q)\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestReorderFlagsFirstParses checks the point of the reordering: that a flag
// written after a positional argument actually takes effect.
func TestReorderFlagsFirstParses(t *testing.T) {
	fs := newTestFlags()
	asJSON := fs.Lookup("json")
	if err := fs.Parse(reorderFlagsFirst(fs, []string{"session-1", "script.js", "-json"})); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if asJSON.Value.String() != "true" {
		t.Error("-json written after the positional arguments was ignored")
	}
	if got := strings.Join(fs.Args(), " "); got != "session-1 script.js" {
		t.Errorf("positional arguments = %q, want %q", got, "session-1 script.js")
	}
}

func TestResolveAPIKeyPrecedence(t *testing.T) {
	// Isolate every source: a real config or a real module directory on the
	// developer's machine would otherwise decide the outcome.
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("BROWSERSCALE_API_KEY", "")

	chdir(t, dir)

	if _, _, err := resolveAPIKey(""); err == nil {
		t.Fatal("expected an error when no source holds a key")
	}

	// 4. the module's config, lowest
	if err := os.MkdirAll("data", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("data", "config.json"), []byte(`{"cloudKey":"module"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	assertKey(t, "", "module", keyFromModule)

	// 3. the saved login beats it
	if _, err := saveCliConfig(cliConfig{APIKey: "login"}); err != nil {
		t.Fatal(err)
	}
	assertKey(t, "", "login", keyFromLogin)

	// 2. the environment beats that
	t.Setenv("BROWSERSCALE_API_KEY", "env")
	assertKey(t, "", "env", keyFromEnv)

	// 1. the flag beats everything
	assertKey(t, "flag", "flag", keyFromFlag)
}

func assertKey(t *testing.T, flagValue, wantKey string, wantSource keySource) {
	t.Helper()
	key, source, err := resolveAPIKey(flagValue)
	if err != nil {
		t.Fatalf("resolveAPIKey(%q): %v", flagValue, err)
	}
	if key != wantKey || source != wantSource {
		t.Errorf("resolveAPIKey(%q) = %q from %q, want %q from %q", flagValue, key, source, wantKey, wantSource)
	}
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{in: "600", want: 600 * time.Second},
		{in: "10m", want: 10 * time.Minute},
		{in: "1h30m", want: 90 * time.Minute},
		{in: "", want: 0},
		{in: "10x", wantErr: true},
		{in: "abc", wantErr: true},
	}
	for _, tc := range cases {
		got, err := parseDuration(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseDuration(%q) = %v, want an error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDuration(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseDuration(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseProxy(t *testing.T) {
	host, port, user, pass, err := parseProxy("proxy.example.com:8080:bob:secret")
	if err != nil {
		t.Fatalf("parseProxy: %v", err)
	}
	if host != "proxy.example.com" || port != 8080 || user != "bob" || pass != "secret" {
		t.Errorf("got %q %d %q %q", host, port, user, pass)
	}

	host, port, user, pass, err = parseProxy("proxy.example.com:8080")
	if err != nil {
		t.Fatalf("parseProxy: %v", err)
	}
	if host != "proxy.example.com" || port != 8080 || user != "" || pass != "" {
		t.Errorf("got %q %d %q %q", host, port, user, pass)
	}

	if _, _, _, _, err := parseProxy("proxy.example.com"); err == nil {
		t.Error("a proxy without a port should be rejected")
	}
	if _, _, _, _, err := parseProxy("proxy.example.com:http"); err == nil {
		t.Error("a non-numeric port should be rejected")
	}
}

// chdir moves into dir for the duration of the test. Go 1.24 has t.Chdir, but
// the CLI still builds for older toolchains in CI.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}
