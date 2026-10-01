package viewer

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/browserscale/browserscale-go/generated"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The DevTools pane talks to the session with the SDK's WebSocket transport,
// and this endpoint is what it connects to. The page frames each call as
//
//	[4B id LE][2B method_len LE][method][protobuf request]
//
// and gets back
//
//	[4B id LE][1B status][protobuf reply | UTF-8 error]
//
// so the page's own client works unchanged. A method without "/" is a control
// frame; the only one is "cancel", which ends the server stream with that id.
//
// The page never holds the API key. Every request message starts with
// session_id = 1 and api_key = 2, and this process appends both fields to each
// payload before forwarding it. Protobuf keeps the last occurrence of a scalar
// field, so whatever the page put there is overridden: it can reach this one
// session and nothing else.

const (
	rpcStatusOK          byte = 0
	rpcStatusError       byte = 1
	rpcStatusStreamMsg   byte = 2
	rpcStatusStreamEnd   byte = 3
	rpcStatusStreamReady byte = 4
)

// A DOM snapshot of a heavy page is the largest reply the pane asks for.
const (
	rpcMaxMessage = 64 << 20
	rpcMaxFrame   = 16 << 20
)

// rpcMethod says how a method is forwarded.
type rpcMethod struct {
	stream bool
}

// rpcMethods lists the calls the pane may make, keyed by full method name.
// A method qualifies only if its request carries session_id and api_key at
// the numbers the injection writes, so a message laid out differently is
// refused rather than sent with the key in some other field.
func rpcMethods() map[string]rpcMethod {
	svc := generated.File_wrc_proto.Services().ByName(protoreflect.Name(serviceShortName()))
	out := map[string]rpcMethod{}
	if svc == nil {
		return out
	}
	methods := svc.Methods()
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		if m.IsStreamingClient() || !hasAuthFields(m.Input()) {
			continue
		}
		out["/"+string(svc.FullName())+"/"+string(m.Name())] = rpcMethod{stream: m.IsStreamingServer()}
	}
	return out
}

func serviceShortName() string {
	name := generated.Browser_ServiceDesc.ServiceName
	return name[strings.LastIndex(name, ".")+1:]
}

func hasAuthFields(msg protoreflect.MessageDescriptor) bool {
	session := msg.Fields().ByNumber(1)
	key := msg.Fields().ByNumber(2)
	return session != nil && session.Name() == "session_id" && session.Kind() == protoreflect.StringKind &&
		key != nil && key.Name() == "api_key" && key.Kind() == protoreflect.StringKind
}

// dialSession opens a gRPC connection to the session's endpoint. "grpcs://"
// means TLS against the system roots; anything else is plaintext, as the SDK
// does it.
func dialSession(grpcURL string) (*grpc.ClientConn, error) {
	target := grpcURL
	creds := insecure.NewCredentials()
	switch {
	case strings.HasPrefix(grpcURL, "grpcs://"):
		target = strings.TrimPrefix(grpcURL, "grpcs://")
		creds = credentials.NewClientTLSFromCert(nil, "")
	case strings.HasPrefix(grpcURL, "grpc://"):
		target = strings.TrimPrefix(grpcURL, "grpc://")
	}
	return grpc.NewClient(target,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(rpcMaxMessage), grpc.MaxCallSendMsgSize(rpcMaxMessage)),
	)
}

// rpcBridge forwards the pane's calls to one session.
type rpcBridge struct {
	cc      *grpc.ClientConn
	methods map[string]rpcMethod
	auth    []byte
	token   func(string) bool

	// done closes every socket when the viewer shuts down. Upgraded
	// connections are hijacked, so http.Server.Shutdown does not reach them.
	done <-chan struct{}
}

func newRPCBridge(ctx context.Context, grpcURL, sessionID, apiKey string, tokenOK func(string) bool) (*rpcBridge, error) {
	cc, err := dialSession(grpcURL)
	if err != nil {
		return nil, fmt.Errorf("could not connect to the session for DevTools: %w", err)
	}
	var auth []byte
	auth = protowire.AppendTag(auth, 1, protowire.BytesType)
	auth = protowire.AppendString(auth, sessionID)
	auth = protowire.AppendTag(auth, 2, protowire.BytesType)
	auth = protowire.AppendString(auth, apiKey)
	return &rpcBridge{cc: cc, methods: rpcMethods(), auth: auth, token: tokenOK, done: ctx.Done()}, nil
}

func (b *rpcBridge) Close() error { return b.cc.Close() }

func (b *rpcBridge) signed(payload []byte) []byte {
	out := make([]byte, 0, len(payload)+len(b.auth))
	out = append(out, payload...)
	return append(out, b.auth...)
}

func (b *rpcBridge) handle(w http.ResponseWriter, r *http.Request) {
	// The token rides in the query because a browser cannot set headers on a
	// WebSocket handshake. The Origin check keeps another site from opening
	// the socket with a token it somehow learned.
	if !b.token(r.URL.Query().Get("t")) {
		http.Error(w, "bad or missing viewer token", http.StatusForbidden)
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || isLoopbackOrigin(origin)
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(rpcMaxFrame)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-b.done:
			_ = conn.Close()
		case <-ctx.Done():
		}
	}()

	sock := &rpcSocket{conn: conn, streams: map[uint32]context.CancelFunc{}}
	defer sock.cancelAll()

	for {
		kind, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if kind != websocket.BinaryMessage || len(raw) < 6 {
			continue
		}
		id := binary.LittleEndian.Uint32(raw[0:4])
		n := int(binary.LittleEndian.Uint16(raw[4:6]))
		if 6+n > len(raw) {
			continue
		}
		method := string(raw[6 : 6+n])
		payload := raw[6+n:]

		if !strings.Contains(method, "/") {
			if method == "cancel" {
				sock.cancel(id)
			}
			continue
		}
		full := "/" + method
		m, ok := b.methods[full]
		if !ok {
			sock.write(id, rpcStatusError, []byte("method not available here: "+method))
			continue
		}
		if m.stream {
			go b.stream(ctx, sock, id, full, b.signed(payload))
			continue
		}
		go func(req []byte) {
			var out rawMessage
			if err := b.cc.Invoke(ctx, full, rawMessage(req), &out, grpc.ForceCodec(rawCodec{})); err != nil {
				sock.write(id, rpcStatusError, []byte(err.Error()))
				return
			}
			sock.write(id, rpcStatusOK, out)
		}(b.signed(payload))
	}
}

func (b *rpcBridge) stream(parent context.Context, sock *rpcSocket, id uint32, full string, req []byte) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	sock.add(id, cancel)
	defer sock.remove(id)

	desc := &grpc.StreamDesc{StreamName: full, ServerStreams: true}
	stream, err := b.cc.NewStream(ctx, desc, full, grpc.ForceCodec(rawCodec{}))
	if err == nil {
		err = stream.SendMsg(rawMessage(req))
	}
	if err == nil {
		err = stream.CloseSend()
	}
	if err == nil {
		// Header returns once the handler is running, which is what makes
		// "ready" mean subscribed: nothing sent after it is missed.
		_, err = stream.Header()
	}
	if err != nil {
		sock.write(id, rpcStatusError, []byte(err.Error()))
		return
	}
	if sock.write(id, rpcStatusStreamReady, nil) != nil {
		return
	}
	for {
		var out rawMessage
		if err := stream.RecvMsg(&out); err != nil {
			switch {
			case ctx.Err() != nil:
				// Cancelled by the page or by the socket closing; nobody is
				// waiting for a frame.
			case errors.Is(err, io.EOF):
				sock.write(id, rpcStatusStreamEnd, nil)
			default:
				sock.write(id, rpcStatusError, []byte(err.Error()))
			}
			return
		}
		if sock.write(id, rpcStatusStreamMsg, out) != nil {
			return
		}
	}
}

// rpcSocket serialises writes, since calls and streams share the socket and
// a WebSocket takes one writer at a time, and tracks the open streams.
type rpcSocket struct {
	conn    *websocket.Conn
	writeMu sync.Mutex

	mu      sync.Mutex
	streams map[uint32]context.CancelFunc
}

func (s *rpcSocket) write(id uint32, status byte, payload []byte) error {
	frame := make([]byte, 5+len(payload))
	binary.LittleEndian.PutUint32(frame[0:4], id)
	frame[4] = status
	copy(frame[5:], payload)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (s *rpcSocket) add(id uint32, cancel context.CancelFunc) {
	s.mu.Lock()
	s.streams[id] = cancel
	s.mu.Unlock()
}

func (s *rpcSocket) remove(id uint32) {
	s.mu.Lock()
	delete(s.streams, id)
	s.mu.Unlock()
}

func (s *rpcSocket) cancel(id uint32) {
	s.mu.Lock()
	cancel := s.streams[id]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *rpcSocket) cancelAll() {
	s.mu.Lock()
	streams := s.streams
	s.streams = map[uint32]context.CancelFunc{}
	s.mu.Unlock()
	for _, cancel := range streams {
		cancel()
	}
}

// rawMessage and rawCodec pass protobuf bytes through gRPC untouched: the page
// already sent wire format, and the reply goes back the same way.
type rawMessage []byte

type rawCodec struct{}

func (rawCodec) Marshal(v any) ([]byte, error) { return v.(rawMessage), nil }

func (rawCodec) Unmarshal(data []byte, v any) error {
	out := v.(*rawMessage)
	*out = append((*out)[:0], data...)
	return nil
}

func (rawCodec) Name() string { return "proto" }
