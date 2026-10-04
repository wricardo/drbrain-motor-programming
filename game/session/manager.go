package session

import (
	"errors"
	"log"
	"sync"

	"github.com/wricardo/drbrain-motor-programming/game/service"
)

// DefaultMaxSessions is used when NewManager is given maxSessions <= 0.
const DefaultMaxSessions = 1000

// Manager is the in-memory registry of live sessions with optional
// write-through persistence. It implements service.SessionStore.
//
// Lock order: ioMu -> Session lock; mu is only ever held briefly and never
// together with a Session lock or while doing file I/O.
type Manager struct {
	p           SessionPersistence
	maxSessions int

	mu       sync.RWMutex
	sessions map[string]*service.Session

	// ioMu serialises persistence writes/deletes so file contents always follow
	// the order in which snapshots were taken, and so a Delete can never be
	// overtaken by a late Persist of the same session.
	ioMu sync.Mutex
}

var _ service.SessionStore = (*Manager)(nil)

// NewManager creates a manager. p may be nil for a memory-only store.
// maxSessions bounds the number of live sessions accepted by Put; <= 0 means
// DefaultMaxSessions.
func NewManager(p SessionPersistence, maxSessions int) *Manager {
	if maxSessions <= 0 {
		maxSessions = DefaultMaxSessions
	}
	return &Manager{p: p, maxSessions: maxSessions, sessions: make(map[string]*service.Session)}
}

// Put registers a new live session. It fails with a LIMIT_REACHED
// *service.Error when maxSessions is reached, ErrInvalidID for a malformed
// id and ErrExists for a duplicate. It does not persist.
func (m *Manager) Put(s *service.Session) error {
	if s == nil || !ValidID(s.ID) {
		return ErrInvalidID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[s.ID]; ok {
		return ErrExists
	}
	if len(m.sessions) >= m.maxSessions {
		return service.Errorf(service.CodeLimit, "session limit reached (%d live sessions); delete one first", m.maxSessions)
	}
	m.sessions[s.ID] = s
	return nil
}

// Get returns the live session pointer.
func (m *Manager) Get(id string) (*service.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Delete removes the session from memory and storage. It returns a NOT_FOUND
// *service.Error if it existed in neither.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	_, found := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()

	if m.p != nil && ValidID(id) {
		m.ioMu.Lock()
		err := m.p.Delete(id)
		m.ioMu.Unlock()
		switch {
		case err == nil:
			found = true
		case errors.Is(err, ErrNotFound):
		default:
			return err
		}
	}
	if !found {
		return service.Errorf(service.CodeNotFound, "session %q not found", id)
	}
	return nil
}

// List returns every live session (live pointers, unspecified order).
func (m *Manager) List() []*service.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*service.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

// Count returns the number of live sessions.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Persist durably writes a snapshot of s. It takes the session lock itself
// (callers MUST NOT hold it). Sessions no longer registered (deleted
// concurrently) are skipped so a late persist cannot resurrect them. With no
// persistence configured it is a no-op.
func (m *Manager) Persist(s *service.Session) error {
	if m.p == nil || s == nil {
		return nil
	}
	m.ioMu.Lock()
	defer m.ioMu.Unlock()

	m.mu.RLock()
	cur := m.sessions[s.ID]
	m.mu.RUnlock()
	if cur != s {
		return nil
	}

	s.Lock()
	snap := s.Clone()
	s.Unlock()
	return m.p.Save(snap)
}

// LoadAll loads every persisted session into memory at startup. Files that
// fail to load (corrupt, unknown schema, invalid map/program/vm) are logged
// and skipped. Loaded sessions are never Playing. The maxSessions limit only
// gates Put; existing persisted sessions are all loaded.
func (m *Manager) LoadAll() error {
	if m.p == nil {
		return nil
	}
	ids, err := m.p.ListAll()
	if err != nil {
		return err
	}
	loaded := make([]*service.Session, 0, len(ids))
	for _, id := range ids {
		s, err := m.p.Load(id)
		if err != nil {
			log.Printf("session: skipping persisted session %s: %v", id, err)
			continue
		}
		s.Playing = false
		loaded = append(loaded, s)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range loaded {
		if _, ok := m.sessions[s.ID]; !ok {
			m.sessions[s.ID] = s
		}
	}
	return nil
}
