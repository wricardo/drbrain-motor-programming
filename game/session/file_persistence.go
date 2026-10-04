package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
)

// SchemaVersion is the current on-disk format version of a session file.
const SchemaVersion = 1

const fileExt = ".json"

// PersistedSession is the JSON form of a session (sessions/<id>.json).
//
// Playing and LastEvent are transient and deliberately not stored: a restored
// session never resumes by itself.
type PersistedSession struct {
	SchemaVersion int              `json:"schema_version"`
	ID            string           `json:"id"`
	DisplayName   string           `json:"display_name"`
	Map           engine.MapConfig `json:"map"` // snapshot taken when the session was created
	Program       engine.Program   `json:"program"`
	VM            engine.VMState   `json:"vm"`
	SpeedMs       int              `json:"speed_ms"`
	Seq           uint64           `json:"seq"`
	CreatedAt     time.Time        `json:"created_at"`
	LastActionAt  time.Time        `json:"last_action_at"`
	Attempts      int              `json:"attempts"`
	BestSteps     int              `json:"best_steps"`
}

// FilePersistence stores one JSON file per session in a directory.
type FilePersistence struct {
	dir string
}

var _ SessionPersistence = (*FilePersistence)(nil)

// NewFilePersistence creates the sessions directory if needed.
func NewFilePersistence(dir string) (*FilePersistence, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions directory: %w", err)
	}
	return &FilePersistence{dir: dir}, nil
}

func (fp *FilePersistence) path(id string) string {
	return filepath.Join(fp.dir, id+fileExt)
}

// Save atomically writes the snapshot to <dir>/<id>.json (temp file in the
// same directory, fsync, rename). s must not be mutated concurrently; the
// Manager passes a private clone.
func (fp *FilePersistence) Save(s *service.Session) (err error) {
	if s == nil {
		return fmt.Errorf("save session: nil session")
	}
	if !ValidID(s.ID) {
		return ErrInvalidID
	}
	if s.Map == nil {
		return fmt.Errorf("save session %s: missing map snapshot", s.ID)
	}
	data, err := json.Marshal(PersistedSession{
		SchemaVersion: SchemaVersion,
		ID:            s.ID,
		DisplayName:   s.DisplayName,
		Map:           s.Map.Config(),
		Program:       s.Program,
		VM:            s.VM,
		SpeedMs:       s.SpeedMs,
		Seq:           s.Seq,
		CreatedAt:     s.CreatedAt,
		LastActionAt:  s.LastActionAt,
		Attempts:      s.Attempts,
		BestSteps:     s.BestSteps,
	})
	if err != nil {
		return fmt.Errorf("marshal session %s: %w", s.ID, err)
	}

	// The temp name never ends in ".json", so ListAll cannot mistake a partial
	// write for a session.
	tmp, err := os.CreateTemp(fp.dir, ".tmp-"+s.ID+"-*")
	if err != nil {
		return fmt.Errorf("create temp file for session %s: %w", s.ID, err)
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("write session %s: %w", s.ID, err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync session %s: %w", s.ID, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close session %s: %w", s.ID, err)
	}
	if err = os.Rename(tmp.Name(), fp.path(s.ID)); err != nil {
		return fmt.Errorf("replace session file %s: %w", s.ID, err)
	}
	if err = syncDir(fp.dir); err != nil {
		return fmt.Errorf("sync dir for session %s: %w", s.ID, err)
	}
	return nil
}

// Load reads and fully validates <dir>/<id>.json. The returned session is
// never Playing.
func (fp *FilePersistence) Load(id string) (*service.Session, error) {
	if !ValidID(id) {
		return nil, ErrInvalidID
	}
	raw, err := os.ReadFile(fp.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read session %s: %w", id, err)
	}
	var ps PersistedSession
	if err := json.Unmarshal(raw, &ps); err != nil {
		return nil, fmt.Errorf("decode session %s: %w", id, err)
	}
	if ps.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("session %s: unsupported schema_version %d (want %d)", id, ps.SchemaVersion, SchemaVersion)
	}
	if ps.ID != id {
		return nil, fmt.Errorf("session file %s contains id %q", id, ps.ID)
	}
	m, err := engine.NewMap(ps.Map)
	if err != nil {
		return nil, fmt.Errorf("session %s: invalid map snapshot: %w", id, err)
	}
	if err := engine.ValidateProgram(m, ps.Program); err != nil {
		return nil, fmt.Errorf("session %s: %w", id, err)
	}
	if err := validateVM(m, &ps.Program, &ps.VM); err != nil {
		return nil, fmt.Errorf("session %s: invalid vm: %w", id, err)
	}
	if ps.SpeedMs < service.MinSpeedMs || ps.SpeedMs > service.MaxSpeedMs {
		ps.SpeedMs = service.DefaultSpeedMs
	}
	last := ps.LastActionAt
	if last.IsZero() {
		last = ps.CreatedAt
	}
	return &service.Session{
		ID: ps.ID, DisplayName: ps.DisplayName, MapID: m.ID, Map: m,
		Program: ps.Program, VM: ps.VM, Playing: false, SpeedMs: ps.SpeedMs,
		Seq: ps.Seq, CreatedAt: ps.CreatedAt, LastActionAt: last,
		Attempts: ps.Attempts, BestSteps: ps.BestSteps,
	}, nil
}

// Delete removes <dir>/<id>.json. It returns ErrNotFound if there is none.
func (fp *FilePersistence) Delete(id string) error {
	if !ValidID(id) {
		return ErrInvalidID
	}
	if err := os.Remove(fp.path(id)); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("remove session %s: %w", id, err)
	}
	if err := syncDir(fp.dir); err != nil {
		return fmt.Errorf("sync dir after removing session %s: %w", id, err)
	}
	return nil
}

// syncDir fsyncs a directory so renames/removals in it are durable.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := d.Sync(); err != nil {
		_ = d.Close()
		return err
	}
	return d.Close()
}

// ListAll returns the ids of every <id>.json file with a valid id; other
// files (temp files, strays) are ignored.
func (fp *FilePersistence) ListAll() ([]string, error) {
	entries, err := os.ReadDir(fp.dir)
	if err != nil {
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), fileExt) {
			continue
		}
		if id := strings.TrimSuffix(e.Name(), fileExt); ValidID(id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// validateVM rejects states that could make the engine index out of range or
// loop, so a hand-edited or corrupted file is skipped instead of crashing a
// runner later.
func validateVM(m *engine.Map, p *engine.Program, vm *engine.VMState) error {
	switch vm.Status {
	case engine.StatusReady, engine.StatusRunning, engine.StatusWon, engine.StatusLost:
	default:
		return fmt.Errorf("unknown status %q", vm.Status)
	}
	switch vm.LossReason {
	case engine.LossNone, engine.LossStepLimit, engine.LossCallDepth, engine.LossProgramEnded:
	default:
		return fmt.Errorf("unknown loss reason %q", vm.LossReason)
	}
	if vm.Facing < engine.FacingUp || vm.Facing > engine.FacingLeft {
		return fmt.Errorf("facing %d out of range", vm.Facing)
	}
	if m.IsRock(vm.Pos.X, vm.Pos.Y) {
		return fmt.Errorf("position (%d,%d) is blocked or out of bounds", vm.Pos.X, vm.Pos.Y)
	}
	if len(vm.Collected) != len(m.Treats) {
		return fmt.Errorf("collected has %d entries, map has %d treats", len(vm.Collected), len(m.Treats))
	}
	left := 0
	for _, c := range vm.Collected {
		if !c {
			left++
		}
	}
	if vm.TreatsLeft != left {
		return fmt.Errorf("treats_left %d does not match collected (%d uncollected)", vm.TreatsLeft, left)
	}
	if vm.Steps < 0 || vm.Steps > m.MaxSteps {
		return fmt.Errorf("steps %d out of range 0..%d", vm.Steps, m.MaxSteps)
	}
	if len(vm.Visited) > vm.Steps+1 {
		return fmt.Errorf("visited has %d entries after %d steps", len(vm.Visited), vm.Steps)
	}
	for _, v := range vm.Visited {
		if m.IsRock(v.X, v.Y) {
			return fmt.Errorf("visited (%d,%d) is blocked or out of bounds", v.X, v.Y)
		}
	}
	if len(vm.CallStack) == 0 || len(vm.CallStack) > m.MaxCallDepth+2 {
		return fmt.Errorf("call stack has %d frames", len(vm.CallStack))
	}
	for i, f := range vm.CallStack {
		var tape []engine.Instruction
		switch {
		case i == 0 && f.Tape != -1:
			return fmt.Errorf("call stack[0] must be main, got tape %d", f.Tape)
		case i == 0:
			tape = p.Main
		case f.Tape < 0 || f.Tape >= len(p.Subs):
			return fmt.Errorf("call stack[%d]: tape %d out of range", i, f.Tape)
		default:
			tape = p.Subs[f.Tape]
		}
		if f.PC < 0 || f.PC > len(tape) {
			return fmt.Errorf("call stack[%d]: pc %d out of range 0..%d", i, f.PC, len(tape))
		}
	}
	return nil
}
