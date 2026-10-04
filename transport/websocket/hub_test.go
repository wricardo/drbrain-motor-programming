package websocket

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wricardo/drbrain-motor-programming/game/service"
)

func snap(id string, seq uint64) *service.Session {
	return &service.Session{ID: id, Seq: seq}
}

func recv(t *testing.T, ch <-chan *service.Session) *service.Session {
	t.Helper()
	select {
	case s, ok := <-ch:
		if !ok {
			t.Fatal("channel closed")
		}
		return s
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for snapshot")
		return nil
	}
}

func noRecv(t *testing.T, ch <-chan *service.Session) {
	t.Helper()
	select {
	case s := <-ch:
		t.Fatalf("unexpected snapshot %+v", s)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestBroadcastRoutesBySession(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a1 := h.SubscribeSession(ctx, "a")
	a2 := h.SubscribeSession(ctx, "a")
	b := h.SubscribeSession(ctx, "b")

	want := snap("a", 1)
	h.Broadcast(want)
	if got := recv(t, a1); got != want {
		t.Fatal("a1 received a different snapshot")
	}
	if got := recv(t, a2); got != want {
		t.Fatal("a2 received a different snapshot")
	}
	noRecv(t, b)

	h.Broadcast(snap("nobody", 1)) // no subscribers: must not block or panic
	h.Broadcast(nil)
}

func TestSlowSubscriberKeepsNewestUpdates(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := h.SubscribeSession(ctx, "s")

	const total = 50
	for seq := uint64(1); seq <= total; seq++ {
		h.Broadcast(snap("s", seq))
	}

	// Nothing was read while 50 updates went by; the buffer holds the 8 newest, in order.
	var got []uint64
	for len(ch) > 0 {
		got = append(got, (<-ch).Seq)
	}
	if len(got) != subscriberBuffer {
		t.Fatalf("buffered %d updates, want %d", len(got), subscriberBuffer)
	}
	for i, seq := range got {
		if want := uint64(total-subscriberBuffer+1) + uint64(i); seq != want {
			t.Fatalf("buffer = %v, want the %d newest in order", got, subscriberBuffer)
		}
	}
	if got[len(got)-1] != total {
		t.Fatal("the newest (terminal) update was lost")
	}
}

func TestSlowSubscriberDoesNotBlockOthers(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	slow := h.SubscribeSession(ctx, "s")
	fast := h.SubscribeSession(ctx, "s")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for seq := uint64(1); seq <= 100; seq++ {
			h.Broadcast(snap("s", seq)) // must never block on slow
		}
	}()
	var last uint64
	for last < 100 {
		last = recv(t, fast).Seq
	}
	<-done
	if n := len(slow); n != subscriberBuffer {
		t.Fatalf("slow buffer = %d, want %d", n, subscriberBuffer)
	}
}

func TestSubscribeClosesOnContextDone(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	ch := h.SubscribeSession(ctx, "s")
	other := h.SubscribeSession(context.Background(), "s")
	if h.SubscriberCount("s") != 2 {
		t.Fatalf("count = %d, want 2", h.SubscriberCount("s"))
	}

	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("received a value instead of close")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel not closed after ctx cancel")
	}
	deadline := time.Now().Add(5 * time.Second)
	for h.SubscriberCount("s") != 1 {
		if time.Now().After(deadline) {
			t.Fatal("cancelled subscription not removed")
		}
		time.Sleep(time.Millisecond)
	}
	h.Broadcast(snap("s", 1)) // the other subscriber is unaffected
	if got := recv(t, other); got.Seq != 1 {
		t.Fatalf("got seq %d", got.Seq)
	}
}

func TestSubscribeWithCancelledContext(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := h.SubscribeSession(ctx, "s")
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("received a value")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel of an already-cancelled subscription never closed")
	}
	if h.SubscriberCount("s") != 0 {
		t.Fatal("subscriber leaked")
	}
}

// TestConcurrentBroadcastSubscribeCancel exercises Broadcast against
// subscribe/cancel churn (run with -race): no send on a closed channel, no deadlock.
func TestConcurrentBroadcastSubscribeCancel(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup
	stop := make(chan struct{})

	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for seq := uint64(1); ; seq++ {
				select {
				case <-stop:
					return
				default:
					h.Broadcast(snap("s", seq))
				}
			}
		}()
	}
	var subs sync.WaitGroup
	for range 8 {
		subs.Add(1)
		go func() {
			defer subs.Done()
			for range 50 {
				ctx, cancel := context.WithCancel(context.Background())
				ch := h.SubscribeSession(ctx, "s")
				for range 3 {
					<-ch
				}
				cancel()
				for range ch { // drains until closed
				}
			}
		}()
	}
	subs.Wait()
	close(stop)
	wg.Wait()
	if n := h.SubscriberCount("s"); n != 0 {
		t.Fatalf("%d subscribers leaked", n)
	}
}

func TestStaleSeqIsDropped(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := h.SubscribeSession(ctx, "s")

	h.Broadcast(snap("s", 5))
	h.Broadcast(snap("s", 3))
	h.Broadcast(snap("s", 5)) // duplicate
	if got := recv(t, ch); got.Seq != 5 {
		t.Fatalf("got seq %d, want 5", got.Seq)
	}
	noRecv(t, ch)
}

func TestStaleFloodCannotEvictNewestSnapshot(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := h.SubscribeSession(ctx, "s")

	const terminal = 100
	h.Broadcast(snap("s", terminal))
	for seq := uint64(1); seq < 50; seq++ {
		h.Broadcast(snap("s", seq))
	}
	if got := recv(t, ch); got.Seq != terminal {
		t.Fatalf("got seq %d, want %d", got.Seq, terminal)
	}
	noRecv(t, ch)
}

func TestSeqTrackingResetsWhenLastSubscriberLeaves(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	ch := h.SubscribeSession(ctx, "s")
	h.Broadcast(snap("s", 9))
	recv(t, ch)
	cancel()
	for range ch {
	}
	h.mu.Lock()
	n := len(h.last)
	h.mu.Unlock()
	if n != 0 {
		t.Fatalf("seq tracking leaked: %d entries", n)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	ch2 := h.SubscribeSession(ctx2, "s")
	h.Broadcast(snap("s", 1)) // e.g. session reset/recreated under same id
	if got := recv(t, ch2); got.Seq != 1 {
		t.Fatalf("got seq %d, want 1", got.Seq)
	}
}
