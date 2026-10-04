package service

import (
	"sync"
	"time"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
)

// Session is one run of one map. The Map is a snapshot taken at creation so
// editing/deleting the source map never invalidates a session.
//
// Live sessions are shared pointers guarded by Lock/Unlock. Anything handed
// outside the service (hub, resolvers) MUST be a Clone.
type Session struct {
	mu sync.Mutex

	ID           string
	DisplayName  string
	MapID        string
	Map          *engine.Map
	Program      engine.Program
	VM           engine.VMState
	Playing      bool
	SpeedMs      int
	Seq          uint64
	CreatedAt    time.Time
	LastActionAt time.Time
	Attempts     int
	BestSteps    int // 0 = never won
	// LastEvent is the most recent step event; transient (not persisted).
	LastEvent *engine.StepEvent
}

// Lock acquires the session mutex.
func (s *Session) Lock() { s.mu.Lock() }

// Unlock releases the session mutex.
func (s *Session) Unlock() { s.mu.Unlock() }

// Clone returns an immutable-by-convention deep copy (Map is shared: it is
// never mutated after creation). Caller must hold the lock if s is live.
func (s *Session) Clone() *Session {
	c := &Session{
		ID: s.ID, DisplayName: s.DisplayName, MapID: s.MapID, Map: s.Map,
		Program: s.Program.Clone(), VM: s.VM.Clone(), Playing: s.Playing, SpeedMs: s.SpeedMs,
		Seq: s.Seq, CreatedAt: s.CreatedAt, LastActionAt: s.LastActionAt,
		Attempts: s.Attempts, BestSteps: s.BestSteps,
	}
	if s.LastEvent != nil {
		ev := *s.LastEvent
		ev.CallStack = append([]engine.Frame(nil), ev.CallStack...)
		c.LastEvent = &ev
	}
	return c
}

// SimulationResult is the outcome of an instant simulation.
type SimulationResult struct {
	Status     engine.Status
	LossReason engine.LossReason
	Steps      int
	FinalState engine.VMState
	Map        *engine.Map
	Events     []engine.StepEvent // only when requested
}

// SessionFilter narrows ListSessions.
type SessionFilter struct {
	MapID string
	Sort  string // "recent" (default, LastActionAt desc) | "created"
	Limit int    // 0 = default 50, max 500
}

// GameService is the single facade used by every transport (GraphQL today).
type GameService interface {
	CreateSession(mapID, displayName string) (*Session, error)
	GetSession(id string) (*Session, error)
	ListSessions(f SessionFilter) ([]*Session, error)
	DeleteSession(id string) error
	RenameSession(id, name string) (*Session, error)

	// SetProgram validates and stores the program and resets the VM to start.
	// Errors: INVALID_PROGRAM, SESSION_PLAYING.
	SetProgram(id string, p engine.Program) (*Session, error)
	// Run starts the server-side ticker (speedMs 50..5000).
	// Errors: SESSION_PLAYING (already), SESSION_TERMINAL, INVALID_ARGUMENT.
	Run(id string, speedMs int) (*Session, error)
	Pause(id string) (*Session, error)
	// Step executes one manual step. Errors: SESSION_PLAYING, SESSION_TERMINAL.
	Step(id string) (*Session, error)
	// Reset puts the VM back to start, keeping the program; stops playback.
	Reset(id string) (*Session, error)
	// Simulate runs instantly without a session.
	Simulate(mapID string, p engine.Program, includeEvents bool) (*SimulationResult, error)

	ListMaps() []*engine.Map
	GetMap(id string) (*engine.Map, error)
	// Admin-gated by the transport; the service itself does not authenticate.
	SaveMap(cfg engine.MapConfig) (*engine.Map, error)
	// CreateMap is SaveMap that fails atomically (INVALID_ARGUMENT) if the id exists.
	CreateMap(cfg engine.MapConfig) (*engine.Map, error)
	DeleteMap(id string) error

	// Shutdown stops all runners and persists sessions.
	Shutdown()
}

// MapStore provides maps (implemented by game/config.Manager).
type MapStore interface {
	List() []*engine.Map
	Get(id string) (*engine.Map, error) // NOT_FOUND error if absent
	Save(cfg engine.MapConfig) (*engine.Map, error)
	Create(cfg engine.MapConfig) (*engine.Map, error) // INVALID_ARGUMENT if id exists
	Delete(id string) error
}

// SessionStore holds live sessions (implemented by game/session.Manager).
// Get returns the live pointer. Persist writes the session durably (no-op for
// memory-only stores); the implementation takes the session lock and clones
// internally, so callers MUST NOT hold the lock when calling Persist.
type SessionStore interface {
	Put(s *Session) error // register a new live session (enforces MAX_SESSIONS)
	Get(id string) (*Session, bool)
	Delete(id string) error
	List() []*Session
	Persist(s *Session) error
}

// Broadcaster publishes immutable snapshots to subscribers
// (implemented by transport/websocket.Hub).
type Broadcaster interface {
	Broadcast(snapshot *Session)
}
