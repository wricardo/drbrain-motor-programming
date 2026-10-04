package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
)

// Run speed bounds (milliseconds per tick) and defaults.
const (
	MinSpeedMs     = 50
	MaxSpeedMs     = 5000
	DefaultSpeedMs = 500

	// DefaultMaxDisplayName is the display-name limit in characters.
	DefaultMaxDisplayName = 40

	// DefaultListLimit and MaxListLimit bound ListSessions.
	DefaultListLimit = 50
	MaxListLimit     = 500
)

// ErrShutdown is returned by operations that would start work after Shutdown.
var ErrShutdown = errors.New("service is shut down")

// Options tunes New. The zero value selects the defaults.
type Options struct {
	// SessionTTL: on New, sessions whose LastActionAt is older than this are
	// deleted. 0 (default) disables sweeping.
	SessionTTL time.Duration
	// MaxDisplayName is the maximum display-name length in characters
	// (default DefaultMaxDisplayName).
	MaxDisplayName int
}

// gameService implements GameService.
//
// Concurrency model: every Session is guarded by its own lock. A session is
// Playing exactly when a runner is attached to it, and both change together
// while the session lock is held (attach/detach take rmu inside it; lock order
// is session -> rmu). Nothing waits for a runner, persists, or broadcasts while
// holding a session lock.
type gameService struct {
	maps     MapStore
	sessions SessionStore
	bc       Broadcaster // may be nil
	opts     Options

	rmu     sync.Mutex // guards runners and closed
	runners map[string]*runner
	closed  bool
}

// New creates the game service. bc may be nil (no subscribers). If
// opts.SessionTTL > 0 expired sessions already present in sessions (e.g. just
// loaded by session.Manager.LoadAll) are swept before New returns.
func New(maps MapStore, sessions SessionStore, bc Broadcaster, opts Options) GameService {
	if opts.MaxDisplayName <= 0 {
		opts.MaxDisplayName = DefaultMaxDisplayName
	}
	svc := &gameService{maps: maps, sessions: sessions, bc: bc, opts: opts, runners: make(map[string]*runner)}
	if opts.SessionTTL > 0 {
		svc.sweepExpired(time.Now().Add(-opts.SessionTTL))
	}
	return svc
}

// sweepExpired deletes sessions idle since before cutoff.
func (svc *gameService) sweepExpired(cutoff time.Time) {
	for _, s := range svc.sessions.List() {
		s.Lock()
		expired := s.LastActionAt.Before(cutoff)
		id := s.ID
		s.Unlock()
		if !expired {
			continue
		}
		if err := svc.DeleteSession(id); err != nil {
			log.Printf("service: sweep expired session %s: %v", id, err)
		}
	}
}

// ---- helpers ---------------------------------------------------------------

func notFound(id string) *Error { return Errorf(CodeNotFound, "session %q not found", id) }

func (svc *gameService) get(id string) (*Session, error) {
	s, ok := svc.sessions.Get(id)
	if !ok {
		return nil, notFound(id)
	}
	return s, nil
}

// persist writes s durably; failures are logged, the in-memory state stays
// authoritative. s must not be locked by the caller.
func (svc *gameService) persist(s *Session) {
	if err := svc.sessions.Persist(s); err != nil {
		log.Printf("service: persist session %s: %v", s.ID, err)
	}
}

func (svc *gameService) broadcast(snap *Session) {
	if svc.bc != nil {
		svc.bc.Broadcast(snap)
	}
}

// touch records a state change: Seq++ and LastActionAt. Lock must be held.
func touch(s *Session) {
	s.Seq++
	s.LastActionAt = time.Now()
}

// initVM returns the start-of-run VM for p. A program with nothing to execute
// would settle straight into LOST/PROGRAM_ENDED; that outcome is only decided
// when the user actually presses Run/Step, so such a VM stays READY.
func initVM(m *engine.Map, p *engine.Program) engine.VMState {
	vm := engine.NewVMState(m)
	vm.Settle(p)
	if vm.Status.Terminal() {
		return engine.NewVMState(m)
	}
	return vm
}

// recordOutcome counts a finished run. Lock must be held and VM terminal.
func recordOutcome(s *Session) {
	s.Attempts++
	if s.VM.Status == engine.StatusWon && (s.BestSteps == 0 || s.VM.Steps < s.BestSteps) {
		s.BestSteps = s.VM.Steps
	}
}

// advance executes one VM step on s and does the bookkeeping shared by ticks
// and manual steps. It reports whether the VM is now terminal. Lock must be
// held and s must not be terminal.
func advance(s *Session) bool {
	ev, err := s.VM.Step(s.Map, &s.Program)
	if err == nil {
		s.LastEvent = &ev
	}
	touch(s)
	if s.VM.Status.Terminal() {
		recordOutcome(s)
		return true
	}
	return false
}

func (svc *gameService) cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n > svc.opts.MaxDisplayName {
		return "", Errorf(CodeInvalidArgument, "display name too long (%d characters, max %d)", n, svc.opts.MaxDisplayName)
	}
	return name, nil
}

func newSessionID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// ---- sessions --------------------------------------------------------------

// CreateSession starts a session on a private snapshot of map mapID with an
// all-EMPTY program.
func (svc *gameService) CreateSession(mapID, displayName string) (*Session, error) {
	name, err := svc.cleanName(displayName)
	if err != nil {
		return nil, err
	}
	m, err := svc.maps.Get(mapID)
	if err != nil {
		return nil, err
	}
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}
	snapMap := m.Clone()
	prog := engine.EmptyProgram(snapMap)
	now := time.Now()
	s := &Session{
		ID: id, DisplayName: name, MapID: snapMap.ID, Map: snapMap,
		Program: prog, VM: initVM(snapMap, &prog), SpeedMs: DefaultSpeedMs,
		Seq: 1, CreatedAt: now, LastActionAt: now,
	}
	if err := svc.sessions.Put(s); err != nil {
		return nil, err
	}
	svc.persist(s)
	s.Lock()
	snap := s.Clone()
	s.Unlock()
	svc.broadcast(snap)
	return snap, nil
}

// GetSession returns a snapshot of the session.
func (svc *gameService) GetSession(id string) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	defer s.Unlock()
	return s.Clone(), nil
}

// ListSessions returns snapshots of the sessions matching f.
func (svc *gameService) ListSessions(f SessionFilter) ([]*Session, error) {
	limit := f.Limit
	switch {
	case limit < 0:
		return nil, Errorf(CodeInvalidArgument, "limit must be >= 0")
	case limit == 0:
		limit = DefaultListLimit
	case limit > MaxListLimit:
		limit = MaxListLimit
	}
	byCreated := false
	switch strings.ToLower(f.Sort) {
	case "", "recent":
	case "created":
		byCreated = true
	default:
		return nil, Errorf(CodeInvalidArgument, "unknown sort %q (want recent or created)", f.Sort)
	}

	type entry struct {
		s   *Session
		id  string
		key time.Time
	}
	var entries []entry
	for _, s := range svc.sessions.List() {
		s.Lock()
		if f.MapID == "" || s.MapID == f.MapID {
			key := s.LastActionAt
			if byCreated {
				key = s.CreatedAt
			}
			entries = append(entries, entry{s: s, id: s.ID, key: key})
		}
		s.Unlock()
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].key.Equal(entries[j].key) {
			return entries[i].key.After(entries[j].key)
		}
		return entries[i].id < entries[j].id
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	out := make([]*Session, len(entries))
	for i, e := range entries {
		e.s.Lock()
		out[i] = e.s.Clone()
		e.s.Unlock()
	}
	return out, nil
}

// DeleteSession stops any runner and removes the session from memory and
// storage.
func (svc *gameService) DeleteSession(id string) error {
	s, live := svc.sessions.Get(id)
	// Remove from the store first: attach refuses sessions that are no longer
	// registered, so no runner can start after this point.
	if err := svc.sessions.Delete(id); err != nil {
		return err
	}
	if live {
		s.Lock()
		r := svc.detach(id)
		s.Playing = false
		s.Unlock()
		r.wait()
	}
	return nil
}

// RenameSession sets the display name.
func (svc *gameService) RenameSession(id, name string) (*Session, error) {
	name, err := svc.cleanName(name)
	if err != nil {
		return nil, err
	}
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	s.DisplayName = name
	touch(s)
	snap := s.Clone()
	s.Unlock()
	svc.persist(s)
	svc.broadcast(snap)
	return snap, nil
}

// SetProgram validates and stores p and resets the VM to the start.
func (svc *gameService) SetProgram(id string, p engine.Program) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	// s.Map is immutable after creation, so it is safe to read unlocked.
	if err := engine.ValidateProgram(s.Map, p); err != nil {
		return nil, Errorf(CodeInvalidProgram, "%v", err)
	}
	prog := p.Clone()

	s.Lock()
	if s.Playing {
		s.Unlock()
		return nil, Errorf(CodeSessionPlaying, "session is playing; pause it before changing the program")
	}
	s.Program = prog
	s.VM = initVM(s.Map, &s.Program)
	s.LastEvent = nil
	touch(s)
	snap := s.Clone()
	s.Unlock()
	svc.persist(s)
	svc.broadcast(snap)
	return snap, nil
}

// Run starts the server-side ticker.
func (svc *gameService) Run(id string, speedMs int) (*Session, error) {
	if speedMs < MinSpeedMs || speedMs > MaxSpeedMs {
		return nil, Errorf(CodeInvalidArgument, "speedMs must be between %d and %d, got %d", MinSpeedMs, MaxSpeedMs, speedMs)
	}
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}

	s.Lock()
	if s.Playing {
		s.Unlock()
		return nil, Errorf(CodeSessionPlaying, "session is already playing")
	}
	if s.VM.Status.Terminal() {
		s.Unlock()
		return nil, Errorf(CodeSessionTerminal, "run is already %s; reset the session first", s.VM.Status)
	}
	s.VM.Settle(&s.Program)
	if s.VM.Status.Terminal() {
		// Nothing to execute: the run ends immediately without a tick.
		recordOutcome(s)
		touch(s)
		snap := s.Clone()
		s.Unlock()
		svc.persist(s)
		svc.broadcast(snap)
		return snap, nil
	}
	r, err := svc.attach(s)
	if err != nil {
		s.Unlock()
		return nil, err
	}
	s.Playing = true
	s.SpeedMs = speedMs
	touch(s)
	snap := s.Clone()
	s.Unlock()
	// Broadcast before starting the goroutine so the first tick's snapshot can
	// never overtake this one. A Pause/Reset/Delete that detaches the runner
	// in between simply waits for the goroutine, which then exits at once.
	svc.broadcast(snap)
	go svc.runLoop(s, r, time.Duration(speedMs)*time.Millisecond)
	return snap, nil
}

// Pause stops playback. It is a no-op for a session that is not playing.
func (svc *gameService) Pause(id string) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	wasPlaying := s.Playing
	r := svc.detach(id)
	if wasPlaying {
		s.Playing = false
		touch(s)
	}
	snap := s.Clone()
	s.Unlock()
	r.wait()
	if wasPlaying {
		svc.persist(s)
		svc.broadcast(snap)
	}
	return snap, nil
}

// Step executes one manual step; it leaves Playing=false.
func (svc *gameService) Step(id string) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	if s.Playing {
		s.Unlock()
		return nil, Errorf(CodeSessionPlaying, "session is playing; pause it before stepping")
	}
	if s.VM.Status.Terminal() {
		s.Unlock()
		return nil, Errorf(CodeSessionTerminal, "run is already %s; reset the session first", s.VM.Status)
	}
	terminal := advance(s)
	snap := s.Clone()
	s.Unlock()
	if terminal {
		svc.persist(s)
	}
	svc.broadcast(snap)
	return snap, nil
}

// Reset stops playback and puts the VM back to the start, keeping the program.
func (svc *gameService) Reset(id string) (*Session, error) {
	s, err := svc.get(id)
	if err != nil {
		return nil, err
	}
	s.Lock()
	r := svc.detach(id)
	s.Playing = false
	s.VM = initVM(s.Map, &s.Program)
	s.LastEvent = nil
	touch(s)
	snap := s.Clone()
	s.Unlock()
	r.wait()
	svc.persist(s)
	svc.broadcast(snap)
	return snap, nil
}

// Simulate runs p on map mapID instantly, without a session. Execution is
// bounded by the map's max_steps.
func (svc *gameService) Simulate(mapID string, p engine.Program, includeEvents bool) (*SimulationResult, error) {
	m, err := svc.maps.Get(mapID)
	if err != nil {
		return nil, err
	}
	if err := engine.ValidateProgram(m, p); err != nil {
		return nil, Errorf(CodeInvalidProgram, "%v", err)
	}
	res := &SimulationResult{Map: m}
	if includeEvents {
		vm := engine.NewVMState(m)
		res.Events, res.FinalState = engine.Simulate(m, p, vm, m.MaxSteps)
	} else {
		res.FinalState = engine.Run(m, p)
	}
	res.Status = res.FinalState.Status
	res.LossReason = res.FinalState.LossReason
	res.Steps = res.FinalState.Steps
	return res, nil
}

// ---- maps ------------------------------------------------------------------

// ListMaps lists all maps.
func (svc *gameService) ListMaps() []*engine.Map { return svc.maps.List() }

// GetMap returns one map.
func (svc *gameService) GetMap(id string) (*engine.Map, error) { return svc.maps.Get(id) }

// SaveMap validates and stores a map. Existing sessions keep their snapshot.
func (svc *gameService) SaveMap(cfg engine.MapConfig) (*engine.Map, error) {
	return svc.maps.Save(cfg)
}

// CreateMap validates and stores a new map; it fails if the id already exists.
func (svc *gameService) CreateMap(cfg engine.MapConfig) (*engine.Map, error) {
	return svc.maps.Create(cfg)
}

// DeleteMap removes a map. Existing sessions keep their snapshot.
func (svc *gameService) DeleteMap(id string) error { return svc.maps.Delete(id) }

// ---- shutdown --------------------------------------------------------------

// Shutdown stops every runner, marks sessions not playing and persists all
// sessions. It is idempotent.
func (svc *gameService) Shutdown() {
	svc.rmu.Lock()
	svc.closed = true
	rs := make([]*runner, 0, len(svc.runners))
	for id, r := range svc.runners {
		rs = append(rs, r)
		delete(svc.runners, id)
	}
	svc.rmu.Unlock()

	for _, r := range rs {
		r.halt()
	}
	for _, r := range rs {
		r.wait()
	}

	for _, s := range svc.sessions.List() {
		s.Lock()
		wasPlaying := s.Playing
		if wasPlaying {
			s.Playing = false
			touch(s)
		}
		snap := s.Clone()
		s.Unlock()
		svc.persist(s)
		if wasPlaying {
			svc.broadcast(snap)
		}
	}
}
