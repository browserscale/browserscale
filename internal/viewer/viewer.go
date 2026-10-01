// Package viewer serves the local page that `browserscale view` opens: a
// loopback HTTP server that hands the browser the stream widget and DevTools,
// answers the handful of signalling calls the widget needs, and carries the
// DevTools pane's calls to the session.
//
// The split is the point. WebRTC media never touches this process — the page
// negotiates with the engine and pulls video from the TURN relay directly — so
// there is no codec, no frame budget and no latency here. What this server
// provides is the one thing the page cannot have: an authenticated route to the
// session. The API key stays in this process and never reaches the page.
package viewer

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	browserscale "github.com/browserscale/browserscale-go"
)

//go:embed assets/viewer.html assets/viewer.js assets/widget.js assets/devtools.js assets/devtools.css
var assets embed.FS

// assets/widget.js is a vendored copy of browserscale-widget's framework-
// agnostic core (its dist/index.js), and assets/devtools.js and devtools.css
// are browserscale-devtools' standalone bundle and stylesheet. They are
// vendored rather than fetched so the binary works offline and so a
// `go install` needs no npm; `node scripts/sync-assets.mjs` refreshes them from
// sibling checkouts and `--check` reports drift.

// Options configures a viewer server.
type Options struct {
	// Browser is an authenticated session connection. The caller owns it.
	Browser *browserscale.CloudBrowser
	// SessionID is shown in the page and its title.
	SessionID string
	// Port to listen on; 0 picks a free one. Always bound to loopback.
	Port int
	// Interactive wires mouse and keyboard through to the page. When false the
	// stream is watch-only.
	Interactive bool
	// Open puts the viewer URL on screen.
	Open bool
	// AppWindow prefers a chromeless window over an ordinary tab, falling back
	// to a tab when no Chromium-based browser is installed.
	AppWindow bool
	// Ready is called once the server is listening, with the viewer URL. The
	// URL is reported through the caller so it decides the format; this package
	// has no business knowing about -json.
	Ready func(url string) error
	// Log receives progress remarks meant for a person.
	Log io.Writer
}

type server struct {
	browser     *browserscale.CloudBrowser
	sessionID   string
	token       string
	interactive bool
	page        []byte
	rpc         *rpcBridge
}

// Serve runs the viewer until ctx is cancelled, then stops the remote encoder.
func Serve(ctx context.Context, opts Options) error {
	token, err := newToken()
	if err != nil {
		return err
	}

	s := &server{
		browser:     opts.Browser,
		sessionID:   opts.SessionID,
		token:       token,
		interactive: opts.Interactive,
	}
	if s.page, err = renderPage(opts.SessionID, token, opts.Interactive); err != nil {
		return err
	}

	rpcCtx, rpcCancel := context.WithCancel(context.Background())
	defer rpcCancel()
	b := opts.Browser
	if s.rpc, err = newRPCBridge(rpcCtx, b.GrpcUrl(), b.SessionId(), b.ApiKey(), s.tokenOK); err != nil {
		return err
	}
	defer s.rpc.Close()

	// Loopback only. This server speaks for an API key, so it has no business
	// being reachable from the network.
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.Port))
	if err != nil {
		return fmt.Errorf("could not listen on 127.0.0.1:%d: %w", opts.Port, err)
	}

	url := fmt.Sprintf("http://%s/?t=%s", ln.Addr().String(), token)
	httpSrv := &http.Server{Handler: s.routes()}

	errc := make(chan error, 1)
	go func() {
		err := httpSrv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errc <- err
	}()

	if opts.Ready != nil {
		if err := opts.Ready(url); err != nil {
			_ = httpSrv.Close()
			return err
		}
	}
	if opts.Open {
		if mode, err := open(url, opts.AppWindow); err != nil {
			fmt.Fprintf(opts.Log, "Could not open a browser (%v). Open the URL above yourself.\n", err)
		} else {
			fmt.Fprintln(opts.Log, describeOpen(mode))
		}
	}
	fmt.Fprintf(opts.Log, "\nStreaming %s. Press Ctrl+C to stop watching.\n", opts.SessionID)
	if opts.Interactive {
		fmt.Fprintf(opts.Log, "Click the picture first, then it takes your mouse and keyboard.\n")
	}

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-errc:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rpcCancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	// Stop the encoder on the way out. The page's own teardown races the tab
	// being closed and usually loses, so this is what actually frees it — and a
	// fresh context, because ours is the one that just got cancelled.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	if err := opts.Browser.StopStream(stopCtx); err != nil {
		fmt.Fprintf(opts.Log, "Note: could not stop the stream: %v\n", err)
	}
	return serveErr
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handlePage)
	// Static, secret-free and side-effect-free, so these need no token: what a
	// foreign page could learn by fetching them is what it could learn by
	// reading our source.
	mux.HandleFunc("/viewer.js", s.handleAsset("assets/viewer.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("/widget.js", s.handleAsset("assets/widget.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("/devtools.js", s.handleAsset("assets/devtools.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("/devtools.css", s.handleAsset("assets/devtools.css", "text/css; charset=utf-8"))

	// The DevTools pane's session channel. It checks the token itself, since a
	// WebSocket handshake is a GET and cannot carry the header guard reads.
	mux.HandleFunc("/ws", s.rpc.handle)

	mux.HandleFunc("/api/ice", s.guard(s.handleIce))
	mux.HandleFunc("/api/start", s.guard(s.handleStart))
	mux.HandleFunc("/api/stop", s.guard(s.handleStop))
	mux.HandleFunc("/api/insert-text", s.guard(s.handleInsertText))
	mux.HandleFunc("/api/selection", s.guard(s.handleSelection))

	return mux
}

func (s *server) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !s.tokenOK(r.URL.Query().Get("t")) {
		http.Error(w, "this URL needs the token browserscale printed", http.StatusForbidden)
		return
	}
	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.Header().Set("cache-control", "no-store")
	_, _ = w.Write(s.page)
}

func (s *server) handleAsset(name, contentType string) http.HandlerFunc {
	body, err := assets.ReadFile(name)
	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", contentType)
		w.Header().Set("cache-control", "no-store")
		_, _ = w.Write(body)
	}
}

// guard is what keeps the rest of the web off this server. Anything the user
// has open in a browser can send requests to 127.0.0.1, so reaching the session
// has to take a secret that only our own page was given.
func (s *server) guard(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		// A custom header cannot be set cross-origin without a preflight, which
		// this server never answers — so the header is a gate before its value
		// is even compared.
		if !s.tokenOK(r.Header.Get("x-viewer-token")) {
			http.Error(w, "bad or missing viewer token", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !isLoopbackOrigin(origin) {
			http.Error(w, "cross-origin requests are not accepted", http.StatusForbidden)
			return
		}
		w.Header().Set("cache-control", "no-store")
		if err := h(w, r); err != nil {
			// A malformed request and a session that would not answer are
			// different problems, and the page reports them to different people:
			// the first is our own bug, the second is the user's to act on.
			status := http.StatusBadGateway
			if errors.As(err, new(badRequestError)) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
		}
	}
}

// isStreamAlreadyActive recognises the engine's refusal to start a second
// stream for one page. It is matched on the message because that is all the
// shape it has: the browser reports it as a CDP error, which arrives here as a
// generic Internal gRPC status.
func isStreamAlreadyActive(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Stream already active")
}

// badRequestError marks an error caused by the request rather than by the
// session behind it.
type badRequestError struct{ error }

func badRequest(format string, args ...any) error {
	return badRequestError{fmt.Errorf(format, args...)}
}

func (s *server) tokenOK(got string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) == 1
}

func (s *server) handleIce(w http.ResponseWriter, r *http.Request) error {
	ice, err := s.browser.GetStreamConfig(r.Context())
	if err != nil {
		return err
	}
	// Shaped as RTCIceServer so the page can hand it straight to the peer.
	out := make([]map[string]any, 0, len(ice))
	for _, srv := range ice {
		entry := map[string]any{"urls": srv.URLs}
		if srv.Username != "" {
			entry["username"] = srv.Username
		}
		if srv.Credential != "" {
			entry["credential"] = srv.Credential
		}
		out = append(out, entry)
	}
	return writeJSON(w, map[string]any{"iceServers": out})
}

func (s *server) handleStart(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		OfferSDP string `json:"offerSdp"`
	}
	if err := readJSON(r, &body); err != nil {
		return badRequest("%s", err)
	}
	if strings.TrimSpace(body.OfferSDP) == "" {
		return badRequest("offerSdp is empty")
	}
	answer, err := s.browser.StartStream(r.Context(), body.OfferSDP)
	if err != nil && isStreamAlreadyActive(err) {
		// The engine allows one stream per page, and a viewer page that went
		// away rarely gets to say so: closing or reloading the tab kills it
		// before its own teardown call goes out. The leftover encoder then
		// blocks the next attempt, which would make reloading the page a
		// permanent failure. Whoever is asking now is the one watching, so take
		// the old stream down and answer them.
		if stopErr := s.browser.StopStream(r.Context()); stopErr == nil {
			answer, err = s.browser.StartStream(r.Context(), body.OfferSDP)
		}
	}
	if err != nil {
		return err
	}
	// null rather than a zero size: the widget leaves pointer input off when it
	// has no coordinate space, which beats mapping every click onto 0,0.
	var viewport any
	if answer.Viewport.Width > 0 && answer.Viewport.Height > 0 {
		viewport = map[string]any{"w": answer.Viewport.Width, "h": answer.Viewport.Height}
	}
	return writeJSON(w, map[string]any{"answerSdp": answer.AnswerSDP, "viewport": viewport})
}

func (s *server) handleStop(w http.ResponseWriter, r *http.Request) error {
	if err := s.browser.StopStream(r.Context()); err != nil {
		return err
	}
	return writeJSON(w, map[string]any{})
}

func (s *server) handleInsertText(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Text string `json:"text"`
	}
	if err := readJSON(r, &body); err != nil {
		return badRequest("%s", err)
	}
	if body.Text == "" {
		return writeJSON(w, map[string]any{})
	}
	if err := s.browser.InsertText(r.Context(), body.Text); err != nil {
		return err
	}
	return writeJSON(w, map[string]any{})
}

func (s *server) handleSelection(w http.ResponseWriter, r *http.Request) error {
	text, err := s.browser.GetSelection(r.Context())
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"text": text})
}

func writeJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("content-type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	// Signalling payloads are small; an SDP offer is a few kilobytes.
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func renderPage(sessionID, token string, interactive bool) ([]byte, error) {
	tmpl, err := template.ParseFS(assets, "assets/viewer.html")
	if err != nil {
		return nil, err
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, struct {
		SessionID   string
		Token       string
		Interactive bool
	}{sessionID, token, interactive})
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate a viewer token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func isLoopbackOrigin(origin string) bool {
	rest, ok := strings.CutPrefix(origin, "http://")
	if !ok {
		return false
	}
	host, _, err := net.SplitHostPort(rest)
	if err != nil {
		host = rest
	}
	return host == "127.0.0.1" || host == "localhost" || host == "[::1]" || host == "::1"
}
