package websocket

import (
	"context"
	"sync"

	"github.com/wricardo/drbrain-motor-programming/game/service"
)

// subscriberBuffer is the per-subscriber channel capacity.
const subscriberBuffer = 8

// Hub fans session snapshots out to subscribers. It implements
// service.Broadcaster. The zero value is not usable; call NewHub.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan *service.Session]struct{} // session id -> subscriber channels
	last map[string]uint64                             // session id -> highest Seq published while subscribed
}

var _ service.Broadcaster = (*Hub)(nil)

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{
		subs: make(map[string]map[chan *service.Session]struct{}),
		last: make(map[string]uint64),
	}
}

// Broadcast publishes snapshot to every subscriber of snapshot.ID. The
// snapshot is shared by all subscribers and MUST be treated as read-only (the
// service only ever hands out clones). Broadcast never blocks: when a
// subscriber's buffer is full its oldest queued snapshot is dropped to make
// room, so the newest update (e.g. the terminal WON/LOST one) is never lost.
// Snapshots whose Seq is not newer than the highest one already published for
// the session are discarded, so a late stale broadcast cannot evict a newer one.
func (h *Hub) Broadcast(snapshot *service.Session) {
	if snapshot == nil {
		return
	}
	// Sending happens under h.mu: it serialises publishers per channel (so the
	// drop-oldest loop below terminates) and excludes close() from the
	// unsubscribe path.
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.subs[snapshot.ID]
	if len(set) == 0 {
		return
	}
	if last, ok := h.last[snapshot.ID]; ok && snapshot.Seq <= last {
		return
	}
	h.last[snapshot.ID] = snapshot.Seq
	for ch := range set {
		publishLatest(ch, snapshot)
	}
}

// publishLatest enqueues v on ch, evicting the oldest queued value while the
// buffer is full. The caller must be the only sender on ch.
func publishLatest(ch chan *service.Session, v *service.Session) {
	for {
		select {
		case ch <- v:
			return
		default:
		}
		select {
		case <-ch: // drop oldest
		default:
		}
	}
}

// SubscribeSession returns a channel that receives a snapshot on every
// Broadcast for sessionID. The channel has a buffer of 8 with latest-wins
// delivery (see Broadcast). It is closed once ctx is done.
func (h *Hub) SubscribeSession(ctx context.Context, sessionID string) <-chan *service.Session {
	ch := make(chan *service.Session, subscriberBuffer)
	h.mu.Lock()
	set := h.subs[sessionID]
	if set == nil {
		set = make(map[chan *service.Session]struct{})
		h.subs[sessionID] = set
	}
	set[ch] = struct{}{}
	h.mu.Unlock()

	// AfterFunc runs the callback once when ctx is done (immediately in its own
	// goroutine if it already is) and costs no goroutine while ctx is live.
	context.AfterFunc(ctx, func() {
		h.mu.Lock()
		if set := h.subs[sessionID]; set != nil {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subs, sessionID)
				delete(h.last, sessionID)
			}
		}
		close(ch)
		h.mu.Unlock()
	})
	return ch
}

// SubscriberCount returns the number of live subscriptions for sessionID.
func (h *Hub) SubscriberCount(sessionID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs[sessionID])
}
