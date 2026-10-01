package viewer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/browserscale/browserscale-go/generated"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func TestRPCMethodsCoverTheService(t *testing.T) {
	methods := rpcMethods()
	desc := generated.Browser_ServiceDesc
	if want := len(desc.Methods) + len(desc.Streams); len(methods) != want {
		t.Fatalf("allow-list has %d methods, the service has %d", len(methods), want)
	}
	for _, sd := range desc.Streams {
		m, ok := methods["/"+desc.ServiceName+"/"+sd.StreamName]
		if !ok || !m.stream {
			t.Errorf("%s should be forwarded as a server stream", sd.StreamName)
		}
	}
	for _, md := range desc.Methods {
		m, ok := methods["/"+desc.ServiceName+"/"+md.MethodName]
		if !ok || m.stream {
			t.Errorf("%s should be forwarded as a unary call", md.MethodName)
		}
	}
}

func TestSignedOverridesWhatThePageSent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b, err := newRPCBridge(ctx, "grpc://127.0.0.1:1", "real-session", "real-key", func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	sent, err := proto.Marshal(&generated.InspectAtPositionRequest{SessionId: "other-session", ApiKey: "guessed", X: 10, Y: 20})
	if err != nil {
		t.Fatal(err)
	}
	var got generated.InspectAtPositionRequest
	if err := proto.Unmarshal(b.signed(sent), &got); err != nil {
		t.Fatal(err)
	}
	if got.SessionId != "real-session" || got.ApiKey != "real-key" {
		t.Fatalf("auth not overridden: session %q key %q", got.SessionId, got.ApiKey)
	}
	if got.X != 10 || got.Y != 20 {
		t.Fatalf("payload changed: x %v y %v", got.X, got.Y)
	}
}

func TestSocketNeedsTokenAndLoopbackOrigin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b, err := newRPCBridge(ctx, "grpc://127.0.0.1:1", "s", "k", func(got string) bool { return got == "secret" })
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	srv := httptest.NewServer(http.HandlerFunc(b.handle))
	defer srv.Close()
	base := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	cases := []struct {
		name   string
		query  string
		origin string
		ok     bool
	}{
		{"no token", "", "http://127.0.0.1:1234", false},
		{"wrong token", "?t=nope", "http://127.0.0.1:1234", false},
		{"foreign origin", "?t=secret", "https://example.com", false},
		{"loopback origin", "?t=secret", "http://127.0.0.1:1234", true},
		{"localhost origin", "?t=secret", "http://localhost:1234", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := http.Header{}
			if c.origin != "" {
				h.Set("Origin", c.origin)
			}
			conn, resp, err := websocket.DefaultDialer.Dial(base+c.query, h)
			if conn != nil {
				conn.Close()
			}
			if c.ok && err != nil {
				t.Fatalf("want upgrade, got %v", err)
			}
			if !c.ok {
				if err == nil {
					t.Fatal("want refusal, got an open socket")
				}
				if resp == nil || resp.StatusCode != http.StatusForbidden {
					t.Fatalf("want 403, got %v", resp)
				}
			}
		})
	}
}

func TestUnknownMethodIsRefused(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b, err := newRPCBridge(ctx, "grpc://127.0.0.1:1", "s", "k", func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	srv := httptest.NewServer(http.HandlerFunc(b.handle))
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?t=x", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	method := "grpc.health.v1.Health/Check"
	frame := []byte{7, 0, 0, 0, byte(len(method)), 0}
	frame = append(frame, method...)
	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		t.Fatal(err)
	}
	_, reply, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if len(reply) < 5 || reply[0] != 7 || reply[4] != rpcStatusError {
		t.Fatalf("want an error frame for id 7, got %v", reply)
	}
}
