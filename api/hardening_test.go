package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/complexity"
	gorillaws "github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2"
	"github.com/wricardo/drbrain-motor-programming/graph"
	"github.com/wricardo/drbrain-motor-programming/graph/generated"
)

// --- subTapeLengths through the HTTP path ---

func TestCreateMapSubTapeLengthsOverHTTP(t *testing.T) {
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("ADMIN_API_KEY", "")
	ts := newTestServer(t, Options{})
	const layout = `["...", ">.*", "..."]`
	cases := []struct {
		name  string
		extra string
		want  string
	}{
		{"omitted", ``, `[10]`},
		{"empty", `, subTapeLengths: []`, `[]`},
		{"explicit", `, subTapeLengths: [3, 4]`, `[3,4]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := `mutation { createMap(map: {id: "fresh", name: "F", layout: ` + layout + tc.extra + `}) { subTapeLengths } }`
			_, out := gql(t, ts.URL, q, nil)
			if out["errors"] != nil {
				t.Fatalf("errors: %v", out)
			}
			got, _ := json.Marshal(out["data"].(map[string]any)["createMap"].(map[string]any)["subTapeLengths"])
			if string(got) != tc.want {
				t.Fatalf("subTapeLengths = %s, want %s", got, tc.want)
			}
		})
	}
}

// --- complexity weighting ---

// Documents mirror frontend/src/lib/queries.ts; keep them in sync when the
// frontend adds heavier queries.
const (
	posSel     = `x y`
	mapSel     = `id name description difficulty width height layout start { x y } startFacing rocks { x y } treats { x y } mainTapeLength subTapeLengths maxSteps maxCallDepth allowRecursion`
	eventSel   = `step instruction tapeIndex slotIndex from { x y } to { x y } facingBefore facingAfter blocked collected { x y } callStack { tape pc } status lossReason`
	sessionSel = `id displayName mapId map { ` + mapSel + ` } program { main subs } vm { pos { x y } facing treatsRemaining { x y } callStack { tape pc } steps status lossReason visited { x y } } playing speedMs seq createdAt lastActionAt attempts bestSteps lastEvent { ` + eventSel + ` }`
	sessionsQ  = `query Sessions($limit: Int, $sort: SessionSort, $mapId: ID) { sessions(sort: $sort, limit: $limit, mapId: $mapId) { id displayName mapId playing createdAt lastActionAt map { name } vm { status steps treatsRemaining { x y } } } }`
)

func queryComplexity(t *testing.T, query string, vars map[string]any) int {
	t.Helper()
	es := generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}, Complexity: complexityRoot()})
	doc, errs := gqlparser.LoadQuery(es.Schema(), query)
	if errs != nil {
		t.Fatalf("parse %q: %v", query, errs)
	}
	return complexity.Calculate(context.Background(), es, doc.Operations[0], vars)
}

func TestFrontendQueriesStayUnderComplexityLimit(t *testing.T) {
	cases := []struct {
		name  string
		query string
		vars  map[string]any
	}{
		{"maps", `query { maps { ` + mapSel + ` } }`, nil},
		{"session", `query($id: ID!) { session(id: $id) { ` + sessionSel + ` } }`, map[string]any{"id": "x"}},
		{"sessions page (limit 500)", sessionsQ, map[string]any{"limit": json.Number("500")}},
		{"sessions home (limit 12)", sessionsQ, map[string]any{"limit": json.Number("12")}},
		{"sessions default limit", sessionsQ, nil},
		{"mutation with full session", `mutation($sessionID: ID!) { run(sessionID: $sessionID) { ` + sessionSel + ` } }`, map[string]any{"sessionID": "x"}},
		{"subscription", `subscription($sessionID: ID!) { sessionUpdated(sessionID: $sessionID) { seq session { ` + sessionSel + ` } event { ` + eventSel + ` } } }`, map[string]any{"sessionID": "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if c := queryComplexity(t, tc.query, tc.vars); c > ComplexityLimit {
				t.Fatalf("complexity %d exceeds limit %d", c, ComplexityLimit)
			}
		})
	}
}

func TestComplexityScalesWithLimit(t *testing.T) {
	sessions := func(limit string) int {
		return queryComplexity(t, `{ sessions(limit: `+limit+`) { id map { name } vm { status } } }`, nil)
	}
	if small, big := sessions("10"), sessions("500"); big <= small*10 {
		t.Fatalf("sessions cost does not grow with limit: limit 10 = %d, limit 500 = %d", small, big)
	}
}

func TestAliasedAbuseIsRejected(t *testing.T) {
	ts := newTestServer(t, Options{})
	alias := func(n int, field string) string {
		var b strings.Builder
		b.WriteString("{ ")
		for i := 0; i < n; i++ {
			b.WriteString("a" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + ": " + field + " ")
		}
		b.WriteString("}")
		return b.String()
	}
	cases := map[string]string{
		"sessions at max limit": alias(3, `sessions(limit: 500) { id map { name } vm { status steps treatsRemaining { x y } } }`),
	}
	for name, q := range cases {
		t.Run(name, func(t *testing.T) {
			status, out := gql(t, ts.URL, q, nil)
			if got := firstErrCode(out); got != "COMPLEXITY_LIMIT_EXCEEDED" {
				t.Fatalf("status=%d code=%q out=%v", status, got, out)
			}
		})
	}
}

// --- CORS ---

func corsRequest(t *testing.T, ts *httptest.Server, method, origin string, extra map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+"/graphql", strings.NewReader(`{"query":"{ maps { id } }"}`))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestCORS(t *testing.T) {
	preflight := map[string]string{"Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "content-type,x-admin-key"}
	t.Run("allowed origin", func(t *testing.T) {
		ts := newTestServer(t, Options{AllowedOrigins: "http://ok.example, http://other.example"})
		resp := corsRequest(t, ts, http.MethodPost, "http://ok.example", nil)
		if resp.StatusCode != 200 || resp.Header.Get("Access-Control-Allow-Origin") != "http://ok.example" {
			t.Fatalf("POST: status %d ACAO %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
		}
		resp = corsRequest(t, ts, http.MethodOptions, "http://ok.example", preflight)
		h := resp.Header
		if resp.StatusCode != http.StatusNoContent || h.Get("Access-Control-Allow-Origin") != "http://ok.example" ||
			h.Get("Access-Control-Allow-Methods") != "POST" || h.Get("Access-Control-Allow-Headers") != "Content-Type, X-Admin-Key" {
			t.Fatalf("preflight: status %d headers %v", resp.StatusCode, h)
		}
	})
	t.Run("disallowed origin", func(t *testing.T) {
		ts := newTestServer(t, Options{AllowedOrigins: "http://ok.example"})
		for _, method := range []string{http.MethodPost, http.MethodOptions} {
			resp := corsRequest(t, ts, method, "http://evil.example", preflight)
			for k := range resp.Header {
				if strings.HasPrefix(k, "Access-Control-Allow") {
					t.Fatalf("%s: unexpected %s header for disallowed origin", method, k)
				}
			}
		}
	})
	t.Run("no origin header", func(t *testing.T) {
		ts := newTestServer(t, Options{AllowedOrigins: "http://ok.example"})
		resp := corsRequest(t, ts, http.MethodPost, "", nil)
		if resp.StatusCode != 200 || resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("status %d ACAO %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
		}
	})
	for _, allowed := range []string{"", "*"} {
		t.Run("allow all "+allowed, func(t *testing.T) {
			ts := newTestServer(t, Options{AllowedOrigins: allowed})
			resp := corsRequest(t, ts, http.MethodOptions, "http://anything.example", preflight)
			if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != "http://anything.example" {
				t.Fatalf("status %d ACAO %q", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

// --- websocket limits ---

func dialWS(t *testing.T, ts *httptest.Server) *gorillaws.Conn {
	t.Helper()
	d := gorillaws.Dialer{Subprotocols: []string{"graphql-transport-ws"}, HandshakeTimeout: 5 * time.Second}
	c, _, err := d.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+"/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestWebsocketInitTimeout(t *testing.T) {
	ts := newTestServer(t, Options{WSInitTimeout: 100 * time.Millisecond})
	c := dialWS(t, ts)
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := c.ReadMessage()
	var ce *gorillaws.CloseError
	if !errors.As(err, &ce) || ce.Code != gorillaws.CloseProtocolError {
		t.Fatalf("want protocol-error close after init timeout, got %v", err)
	}
}

func TestWebsocketInitWithinTimeoutIsAcked(t *testing.T) {
	ts := newTestServer(t, Options{WSInitTimeout: 5 * time.Second})
	c := dialWS(t, ts)
	if err := c.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"connection_init"}`)); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := c.ReadMessage()
	if err != nil || !strings.Contains(string(msg), "connection_ack") {
		t.Fatalf("ack = %q, %v", msg, err)
	}
}

func TestWebsocketOversizedMessageClosesConnection(t *testing.T) {
	ts := newTestServer(t, Options{})
	c := dialWS(t, ts)
	if err := c.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"connection_init"}`)); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := c.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	// A syntactically valid ping that only the size limit can reject: without
	// it the server would answer pong.
	big := `{"type":"ping","payload":{"pad":"` + strings.Repeat("a", MaxRequestBodyBytes) + `"}}`
	// The write itself may fail once the server hangs up mid-frame.
	_ = c.WriteMessage(gorillaws.TextMessage, []byte(big))
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				t.Fatal("connection still open after oversized message")
			}
			return // closed, as required
		}
		if strings.Contains(string(msg), `"pong"`) {
			t.Fatalf("oversized message was processed: %s", msg)
		}
	}
}

func TestWSFrameGuard(t *testing.T) {
	// frame builds a masked client frame header followed by zero payload.
	frame := func(fin bool, opcode byte, payload int) []byte {
		b0 := opcode
		if fin {
			b0 |= 0x80
		}
		var h []byte
		switch {
		case payload < 126:
			h = []byte{b0, 0x80 | byte(payload)}
		case payload < 1<<16:
			h = []byte{b0, 0x80 | 126, byte(payload >> 8), byte(payload)}
		default:
			h = []byte{b0, 0x80 | 127, 0, 0, 0, 0, byte(payload >> 24), byte(payload >> 16), byte(payload >> 8), byte(payload)}
		}
		h = append(h, 1, 2, 3, 4) // mask key
		return append(h, make([]byte, payload)...)
	}
	cat := func(parts ...[]byte) []byte {
		var out []byte
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	const limit = 1000
	cases := []struct {
		name     string
		stream   []byte
		tooBig   bool
		byteWise bool
	}{
		{"exactly limit", frame(true, 1, limit), false, false},
		{"one over", frame(true, 1, limit+1), true, false},
		{"fragments sum over", cat(frame(false, 1, 600), frame(true, 0, 401)), true, false},
		{"fragments sum at limit", cat(frame(false, 1, 600), frame(true, 0, 400)), false, false},
		{"counter resets between messages", cat(frame(true, 1, 900), frame(true, 1, 900)), false, false},
		{"control frames not counted", cat(frame(false, 1, 900), frame(true, 9, 100), frame(true, 0, 100)), false, false},
		{"header split across reads", cat(frame(true, 2, 2000)), true, true},
		{"empty frames", cat(frame(true, 1, 0), frame(true, 1, 0)), false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := wsFrameGuard{limit: limit}
			var err error
			if tc.byteWise {
				for i := range tc.stream {
					if err = g.feed(tc.stream[i : i+1]); err != nil {
						break
					}
				}
			} else {
				err = g.feed(tc.stream)
			}
			if (err != nil) != tc.tooBig {
				t.Fatalf("err = %v, want tooBig=%v", err, tc.tooBig)
			}
		})
	}
}
