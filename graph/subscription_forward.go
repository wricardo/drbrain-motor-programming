package graph

import (
	"context"

	"github.com/wricardo/drbrain-motor-programming/game/service"
	"github.com/wricardo/drbrain-motor-programming/graph/model"
)

// forwardSessionUpdates emits the initial snapshot, then converts hub updates
// until ctx is done. Every send also selects on ctx.Done: gqlgen stops reading
// the returned channel once the subscription ends, so an unconditional send
// would block this goroutine (and the snapshots it holds) forever. Updates
// whose seq is not newer than the last emitted one are dropped (the initial
// snapshot may race with a hub update carrying the same state).
func forwardSessionUpdates(ctx context.Context, initial *service.Session, updates <-chan *service.Session, buffer int) <-chan *model.SessionUpdate {
	out := make(chan *model.SessionUpdate, buffer)
	go func() {
		defer close(out)
		last := initial.Seq
		select {
		case out <- toSessionUpdate(initial):
		case <-ctx.Done():
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case u, ok := <-updates:
				if !ok {
					return
				}
				if u.Seq <= last {
					continue
				}
				last = u.Seq
				select {
				case out <- toSessionUpdate(u):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
