package api

import (
	"bufio"
	"encoding/binary"
	"errors"
	"math"
	"net"
	"net/http"
)

// errWSMessageTooLarge is returned from reads once a client websocket message
// exceeds the configured limit.
var errWSMessageTooLarge = errors.New("websocket: message exceeds read limit")

// limitWebsocketMessages closes upgraded GraphQL websocket connections whose
// client messages exceed MaxRequestBodyBytes. gqlgen's Websocket transport
// owns the *gorilla.Conn and exposes no hook for SetReadLimit, so the limit is
// enforced one layer lower: the hijacked net.Conn parses client frame headers
// and refuses to read further once a message (all its fragments) is too big.
func limitWebsocketMessages(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "" {
			w = &limitedHijacker{ResponseWriter: w, limit: MaxRequestBodyBytes}
		}
		next.ServeHTTP(w, r)
	})
}

// limitedHijacker wraps the hijacked connection in a wsLimitConn.
type limitedHijacker struct {
	http.ResponseWriter
	limit int64
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (h *limitedHijacker) Unwrap() http.ResponseWriter { return h.ResponseWriter }

func (h *limitedHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := h.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("api: response does not implement http.Hijacker")
	}
	conn, brw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}
	lc := &wsLimitConn{Conn: conn, guard: wsFrameGuard{limit: h.limit}}
	// Reads must go through lc, so replace the hijacked reader. The upgrader
	// rejects pre-handshake buffered data, so nothing is lost.
	return lc, bufio.NewReadWriter(bufio.NewReader(lc), brw.Writer), nil
}

// wsLimitConn is a net.Conn that fails reads once the guard trips.
type wsLimitConn struct {
	net.Conn
	guard wsFrameGuard
}

func (c *wsLimitConn) Read(p []byte) (int, error) {
	if c.guard.err != nil {
		return 0, c.guard.err
	}
	n, err := c.Conn.Read(p)
	if gerr := c.guard.feed(p[:n]); gerr != nil {
		_ = c.Conn.Close()
		return 0, gerr
	}
	return n, err
}

// wsFrameGuard incrementally parses the RFC 6455 frame headers of the client
// byte stream and tracks the payload size of the current message. It does not
// buffer payloads. Compression is not negotiated, so lengths are wire lengths.
type wsFrameGuard struct {
	limit     int64
	hdr       [14]byte // longest header: 2 + 8 (ext length) + 4 (mask)
	hdrLen    int
	remaining int64 // payload bytes left in the current frame
	msgLen    int64 // data payload bytes of the in-progress message
	endsMsg   bool  // current frame is a final data frame
	err       error
}

func (g *wsFrameGuard) feed(b []byte) error {
	for len(b) > 0 && g.err == nil {
		if g.remaining > 0 {
			n := min(g.remaining, int64(len(b)))
			g.remaining -= n
			b = b[n:]
			if g.remaining == 0 {
				g.frameDone()
			}
			continue
		}
		g.hdr[g.hdrLen] = b[0]
		g.hdrLen++
		b = b[1:]
		if g.hdrLen >= 2 && g.hdrLen == g.headerSize() {
			g.startFrame()
		}
	}
	return g.err
}

// headerSize is the full header length implied by the first two bytes.
func (g *wsFrameGuard) headerSize() int {
	n := 2
	switch g.hdr[1] & 0x7f {
	case 126:
		n += 2
	case 127:
		n += 8
	}
	if g.hdr[1]&0x80 != 0 {
		n += 4
	}
	return n
}

func (g *wsFrameGuard) startFrame() {
	fin := g.hdr[0]&0x80 != 0
	control := g.hdr[0]&0x0f >= 8
	var plen uint64
	switch l := g.hdr[1] & 0x7f; l {
	case 126:
		plen = uint64(binary.BigEndian.Uint16(g.hdr[2:4]))
	case 127:
		plen = binary.BigEndian.Uint64(g.hdr[2:10])
	default:
		plen = uint64(l)
	}
	g.hdrLen = 0
	if plen > math.MaxInt64 {
		g.err = errWSMessageTooLarge
		return
	}
	g.remaining = int64(plen)
	g.endsMsg = fin && !control
	if control {
		return // control frames are ≤125 bytes and not part of a message
	}
	if g.remaining > g.limit-g.msgLen {
		g.err = errWSMessageTooLarge
		return
	}
	g.msgLen += g.remaining
	if g.remaining == 0 {
		g.frameDone()
	}
}

func (g *wsFrameGuard) frameDone() {
	if g.endsMsg {
		g.msgLen = 0
	}
}
