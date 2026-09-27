package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	browserscale "github.com/browserscale/browserscale-go"
)

// The API key a fleet command uses, and where it is allowed to come from.
//
// Four sources, most explicit first:
//
//  1. -key on the command
//  2. BROWSERSCALE_API_KEY in the environment
//  3. the key `browserscale login` saved
//  4. cloudKey in ./data/config.json, when run inside a module
//
// The order is what makes the last one safe to have. A module's config is the
// key you are already using for that project, so picking it up means `list` and
// `run` work in a module directory with no setup at all. But it is also the
// wrong key as soon as you have two projects, so anything stated explicitly —
// including a shell that exported a key on purpose — has to win over it.
//
// Reading from a module's config never writes to it: `login` only ever touches
// the CLI's own file.

// cliConfig is what `login` persists.
type cliConfig struct {
	APIKey string `json:"apiKey"`
	// Endpoint points the CLI at a non-public deployment. Rarely set; the
	// BROWSERSCALE_API_URL environment variable covers the one-off case.
	Endpoint string `json:"endpoint,omitempty"`
}

// keySource names where a resolved key came from, for error messages worth
// acting on: "invalid key" is only useful if you know which key was used.
type keySource string

const (
	keyFromFlag   keySource = "-key flag"
	keyFromEnv    keySource = "BROWSERSCALE_API_KEY"
	keyFromLogin  keySource = "browserscale login"
	keyFromModule keySource = "data/config.json"
)

// configPath is where `login` stores the key: %AppData%\browserscale\config.json
// on Windows, $XDG_CONFIG_HOME/browserscale/config.json elsewhere.
func configPath() (string, error) {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("AppData"); dir != "" {
			return filepath.Join(dir, "browserscale", "config.json"), nil
		}
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "browserscale", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate a config directory: %w", err)
	}
	return filepath.Join(home, ".config", "browserscale", "config.json"), nil
}

// loadCliConfig reads the saved config. A missing file is not an error — it just
// means nobody has logged in.
func loadCliConfig() (cliConfig, error) {
	path, err := configPath()
	if err != nil {
		return cliConfig{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cliConfig{}, nil
	}
	if err != nil {
		return cliConfig{}, err
	}
	var cfg cliConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cliConfig{}, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	return cfg, nil
}

func saveCliConfig(cfg cliConfig) (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	// 0600: the file holds a credential, and on a shared machine the default
	// umask would leave it world-readable.
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// moduleKey reads cloudKey out of a module's config, if the working directory is
// one. Errors are swallowed: a project with a half-written config should fall
// through to "no key found" rather than fail a command that never needed it.
func moduleKey() string {
	data, err := os.ReadFile(filepath.Join("data", "config.json"))
	if err != nil {
		return ""
	}
	var cfg struct {
		CloudKey string `json:"cloudKey"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.CloudKey)
}

// resolveAPIKey finds the key to use and says where it came from.
func resolveAPIKey(flagKey string) (string, keySource, error) {
	if key := strings.TrimSpace(flagKey); key != "" {
		return key, keyFromFlag, nil
	}
	if key := strings.TrimSpace(os.Getenv("BROWSERSCALE_API_KEY")); key != "" {
		return key, keyFromEnv, nil
	}
	cfg, err := loadCliConfig()
	if err != nil {
		return "", "", err
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		return key, keyFromLogin, nil
	}
	if key := moduleKey(); key != "" {
		return key, keyFromModule, nil
	}
	return "", "", errors.New("no API key. Run `browserscale login`, set BROWSERSCALE_API_KEY, or pass -key")
}

// applyEndpoint points the SDK at a non-default deployment, if one is
// configured. The environment wins over the saved config, so a single command
// can be aimed at a local bserver without editing anything.
func applyEndpoint() {
	if url := strings.TrimSpace(os.Getenv("BROWSERSCALE_API_URL")); url != "" {
		browserscale.SetApiEndpoint(strings.TrimSuffix(url, "/"))
		return
	}
	cfg, err := loadCliConfig()
	if err != nil {
		return
	}
	if cfg.Endpoint != "" {
		browserscale.SetApiEndpoint(strings.TrimSuffix(cfg.Endpoint, "/"))
	}
}

// currentEndpoint reports which API the commands will talk to, and why. The
// counterpart to showing the key: pointing the CLI at a local bserver and
// forgetting is the other way to be puzzled by the answers you get.
func currentEndpoint() string {
	if url := strings.TrimSpace(os.Getenv("BROWSERSCALE_API_URL")); url != "" {
		return strings.TrimSuffix(url, "/") + "  (from BROWSERSCALE_API_URL)"
	}
	if cfg, err := loadCliConfig(); err == nil && cfg.Endpoint != "" {
		return cfg.Endpoint + "  (from `login -endpoint`)"
	}
	return browserscale.ApiEndpoint + "  (default)"
}

func runLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	var (
		key      = fs.String("key", "", "API key to save (read from stdin when omitted)")
		endpoint = fs.String("endpoint", "", "base URL of a private deployment (default: the public API)")
		show     = fs.Bool("show", false, "print where the key currently comes from and exit")
	)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage: browserscale login [flags]
       browserscale login -endpoint http://localhost:8004   (keeps the saved key)
       browserscale login -endpoint ""                      (back to the public API)

Saves an API key so the fleet commands (list, rent, run, stop) can use it
without a flag. The key is written to this machine's config with owner-only
permissions; it is not sent anywhere by this command.

-endpoint points the CLI at a private or local deployment and sticks. For a
one-off — testing against a bserver you are running — set BROWSERSCALE_API_URL
instead, which wins over the saved value and disappears with the shell.

Flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *show {
		// A missing key is one of the things worth reporting here, not a reason to
		// refuse: "which server am I even pointed at" is exactly the question you
		// ask before you have logged in.
		key := "none"
		if resolved, source, err := resolveAPIKey(""); err == nil {
			key = fmt.Sprintf("%s  (from %s)", maskKey(resolved), source)
		}
		path, _ := configPath()
		fmt.Printf("key      %s\n", key)
		fmt.Printf("endpoint %s\n", currentEndpoint())
		fmt.Printf("config   %s\n", path)
		return nil
	}

	cfg, err := loadCliConfig()
	if err != nil {
		return err
	}

	// Changing only the endpoint is its own errand — pointing at a local bserver
	// and back again. Prompting for a key that is already saved would make that
	// mean re-pasting a credential to change an unrelated setting.
	endpointOnly := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "endpoint" {
			endpointOnly = true
		}
	})
	endpointOnly = endpointOnly && strings.TrimSpace(*key) == "" && cfg.APIKey != ""

	value := strings.TrimSpace(*key)
	if value == "" && !endpointOnly {
		// Read rather than prompt-and-echo: this is a credential, and echoing it
		// leaves it in the scrollback. `browserscale login < key.txt` and a
		// pipe both work, and an interactive paste ends at the first newline.
		fmt.Fprint(os.Stderr, "Paste your API key: ")
		read, err := readLine()
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)
		value = strings.TrimSpace(read)
	}
	if value == "" && !endpointOnly {
		return errors.New("no key given")
	}

	if value != "" {
		cfg.APIKey = value
	}
	if *endpoint != "" {
		cfg.Endpoint = strings.TrimSuffix(*endpoint, "/")
	} else if endpointOnly {
		// An explicit empty -endpoint is how you go back to the public API, so it
		// has to clear the setting rather than being read as "leave it alone".
		cfg.Endpoint = ""
	}
	path, err := saveCliConfig(cfg)
	if err != nil {
		return err
	}

	if endpointOnly {
		target := cfg.Endpoint
		if target == "" {
			target = "the public API"
		}
		fmt.Printf("Now using %s (key unchanged), saved to %s\n", target, path)
	} else {
		fmt.Printf("Saved %s to %s\n", maskKey(cfg.APIKey), path)
	}
	if source := shadowedBy(); source != "" {
		// Saving a key that something more explicit overrides is a trap worth
		// naming at once, rather than leaving the user to wonder why `list`
		// still answers for the wrong account.
		fmt.Printf("\nNote: %s is set and takes precedence, so commands will keep using that key.\n", source)
	}
	return nil
}

// shadowedBy reports a key source that outranks the saved config, if any.
func shadowedBy() string {
	if strings.TrimSpace(os.Getenv("BROWSERSCALE_API_KEY")) != "" {
		return "BROWSERSCALE_API_KEY"
	}
	return ""
}

// maskKey shows enough of a key to recognise which one it is, and not enough to
// use it — these lines end up in terminal scrollback and pasted bug reports.
func maskKey(key string) string {
	if len(key) <= 8 {
		return "…"
	}
	return key[:5] + "…" + key[len(key)-3:]
}

func readLine() (string, error) {
	var line []byte
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				break
			}
			if buf[0] != '\r' {
				line = append(line, buf[0])
			}
		}
		if err != nil {
			if len(line) > 0 {
				break
			}
			return "", err
		}
	}
	return string(line), nil
}
