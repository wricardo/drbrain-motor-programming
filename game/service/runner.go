package service

import (
	"sync"
	"time"
)

// runner drives one Playing session: a goroutine that steps the VM on a
// ticker until it is halted or the run reaches a terminal state.
//
// Ownership rules (see gameService):
//   - A runner is registered in gameService.runners exactly while its session
//     is Playing; attach/detach (and the terminal tick) run under the session
//     lock.
//   - halt is non-blocking and idempotent. wait blocks until the goroutine has
//     fully exited, so it MUST be called without any session lock held.
//   - After halt the goroutine never touches the VM again: every tick re-checks
//     stopped() under the session lock.
type runner struct {
	id   string
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

func newRunner(id string) *runner {
	return &runner{id: id, stop: make(chan struct{}), done: make(chan struct{})}
}

// halt asks the goroutine to exit. It never blocks and is nil-safe.
func (r *runner) halt() {
	if r != nil {
		r.once.Do(func() { close(r.stop) })
	}
}

// wait blocks until the goroutine has exited. It is nil-safe.
func (r *runner) wait() {
	if r != nil {
		<-r.done
	}
}

func (r *runner) stopped() bool {
	select {
	case <-r.stop:
		return true
	default:
		return false
	}
}

// attach registers a new runner for s. The caller holds s's lock. It fails if
// the service is shut down or s is no longer the registered session for its id
// (deleted concurrently), which is what guarantees no runner can outlive
// DeleteSession.
func (svc *gameService) attach(s *Session) (*runner, error) {
	svc.rmu.Lock()
	defer svc.rmu.Unlock()
	if svc.closed {
		return nil, ErrShutdown
	}
	if cur, ok := svc.sessions.Get(s.ID); !ok || cur != s {
		return nil, notFound(s.ID)
	}
	r := newRunner(s.ID)
	svc.runners[s.ID] = r
	return r, nil
}

// detach unregisters and halts the runner for id, if any, and returns it so
// the caller can wait() after releasing the session lock. This is the single
// stop path used by Pause, Reset and DeleteSession (Shutdown does the same in
// bulk).
func (svc *gameService) detach(id string) *runner {
	svc.rmu.Lock()
	r := svc.runners[id]
	delete(svc.runners, id)
	svc.rmu.Unlock()
	r.halt()
	return r
}

// unregister removes r from the registry if it is still the current runner.
func (svc *gameService) unregister(r *runner) {
	svc.rmu.Lock()
	if svc.runners[r.id] == r {
		delete(svc.runners, r.id)
	}
	svc.rmu.Unlock()
}

// runLoop is the runner goroutine.
func (svc *gameService) runLoop(s *Session, r *runner, every time.Duration) {
	defer close(r.done)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-t.C:
		}
		if !svc.tick(s, r) {
			return
		}
	}
}

// tick performs one step. It reports whether the runner should keep going.
func (svc *gameService) tick(s *Session, r *runner) bool {
	s.Lock()
	if r.stopped() || !s.Playing {
		// Pause/Reset/Delete/Shutdown already updated the session.
		s.Unlock()
		return false
	}
	terminal := advance(s)
	if terminal {
		s.Playing = false
		svc.unregister(r)
	}
	snap := s.Clone()
	s.Unlock()

	if terminal {
		svc.persist(s)
	}
	svc.broadcast(snap)
	return !terminal
}
