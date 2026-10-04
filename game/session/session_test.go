package session

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
)

const (
	idA = "00112233445566aa"
	idB = "00112233445566bb"
	idC = "00112233445566cc"
)

var corridorCfg = engine.MapConfig{
	ID: "corridor", Name: "Corridor", Layout: []string{".....", ">...*", "....."},
	MainTapeLength: 6, SubTapeLengths: []int{2}, MaxSteps: 30, MaxCallDepth: 8,
}

func init() {
	// Skipped/corrupt-file tests log on purpose.
	log.SetOutput(io.Discard)
}

func mustMap(t testing.TB, cfg engine.MapConfig) *engine.Map {
	t.Helper()
	m, err := engine.NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// fixture builds a session two steps into a winning-bound run, with every
// persisted field set to a distinguishable value.
func fixture(t testing.TB, id string) *service.Session {
	t.Helper()
	m := mustMap(t, corridorCfg)
	prog := engine.EmptyProgram(m)
	for i := range 4 {
		prog.Main[i] = engine.MoveForward
	}
	vm := engine.NewVMState(m)
	vm.Settle(&prog)
	var last engine.StepEvent
	for range 2 {
		ev, err := vm.Step(m, &prog)
		if err != nil {
			t.Fatal(err)
		}
		last = ev
	}
	return &service.Session{
		ID: id, DisplayName: "fixture", MapID: m.ID, Map: m, Program: prog, VM: vm,
		Playing: true, SpeedMs: 120, Seq: 17,
		CreatedAt:    time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC),
		LastActionAt: time.Date(2026, 1, 2, 4, 4, 5, 6, time.UTC),
		Attempts:     3, BestSteps: 5, LastEvent: &last,
	}
}

func newFile(t testing.TB) (*FilePersistence, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions")
	fp, err := NewFilePersistence(dir)
	if err != nil {
		t.Fatal(err)
	}
	return fp, dir
}

func TestValidID(t *testing.T) {
	good := []string{idA, "0123456789abcdef", "ffffffffffffffff"}
	bad := []string{
		"", "0123", "0123456789abcde", "0123456789abcdef0", "0123456789ABCDEF", "0123456789abcdeg",
		"../0123456789abcd", "..", ".", "0123456789abcd/.", "0123456789abc\\..", " 123456789abcdef", "0123456789abcdef\n",
		"/etc/passwd", "0123456789abcdef.json",
	}
	for _, id := range good {
		if !ValidID(id) {
			t.Errorf("ValidID(%q) = false", id)
		}
	}
	for _, id := range bad {
		if ValidID(id) {
			t.Errorf("ValidID(%q) = true", id)
		}
	}
}

func TestFileRoundTrip(t *testing.T) {
	fp, dir := newFile(t)
	want := fixture(t, idA)
	if err := fp.Save(want); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, idA+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	if generic["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %v", generic["schema_version"])
	}
	for _, k := range []string{"id", "display_name", "map", "program", "vm", "speed_ms", "seq", "created_at", "last_action_at", "attempts", "best_steps"} {
		if _, ok := generic[k]; !ok {
			t.Errorf("persisted file lacks %q", k)
		}
	}
	for _, k := range []string{"playing", "last_event"} {
		if _, ok := generic[k]; ok {
			t.Errorf("persisted file contains transient field %q", k)
		}
	}

	got, err := fp.Load(idA)
	if err != nil {
		t.Fatal(err)
	}
	if got.Playing {
		t.Fatal("loaded session is Playing")
	}
	if got.LastEvent != nil {
		t.Fatal("LastEvent is transient and must not be restored")
	}
	if got.ID != idA || got.DisplayName != "fixture" || got.MapID != "corridor" || got.SpeedMs != 120 ||
		got.Seq != 17 || got.Attempts != 3 || got.BestSteps != 5 {
		t.Fatalf("scalar fields differ: %+v", got)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || !got.LastActionAt.Equal(want.LastActionAt) {
		t.Fatalf("timestamps differ: %v %v", got.CreatedAt, got.LastActionAt)
	}
	if !reflect.DeepEqual(got.Program, want.Program) {
		t.Fatalf("program differs:\n got %+v\nwant %+v", got.Program, want.Program)
	}
	if !reflect.DeepEqual(got.VM, want.VM) {
		t.Fatalf("vm differs:\n got %+v\nwant %+v", got.VM, want.VM)
	}
	if !reflect.DeepEqual(got.Map.Config(), want.Map.Config()) {
		t.Fatalf("map snapshot differs: %+v", got.Map.Config())
	}

	// The restored VM must be steppable exactly like the original.
	for !want.VM.Status.Terminal() {
		w, werr := want.VM.Step(want.Map, &want.Program)
		g, gerr := got.VM.Step(got.Map, &got.Program)
		if werr != nil || gerr != nil || !reflect.DeepEqual(w, g) {
			t.Fatalf("restored VM diverged: %+v vs %+v (%v/%v)", g, w, gerr, werr)
		}
	}
	if got.VM.Status != engine.StatusWon {
		t.Fatalf("restored run ended %s, want WON", got.VM.Status)
	}
}

func TestFileRoundTripZeroSubMap(t *testing.T) {
	fp, _ := newFile(t)
	m := mustMap(t, engine.MapConfig{
		ID: "nosubs", Name: "n", Layout: []string{".....", ">...*", "....."},
		MainTapeLength: 4, SubTapeLengths: []int{}, MaxSteps: 30,
	})
	prog := engine.EmptyProgram(m)
	s := &service.Session{ID: idA, MapID: m.ID, Map: m, Program: prog, VM: engine.NewVMState(m), SpeedMs: 500, Seq: 1}
	if err := fp.Save(s); err != nil {
		t.Fatal(err)
	}
	got, err := fp.Load(idA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Map.SubTapeLengths) != 0 || len(got.Program.Subs) != 0 {
		t.Fatalf("zero-sub map came back with subs %v / %d", got.Map.SubTapeLengths, len(got.Program.Subs))
	}
}

func TestFileSaveIsAtomicAndListIgnoresStrays(t *testing.T) {
	fp, dir := newFile(t)
	s := fixture(t, idA)
	for range 3 {
		s.Seq++
		if err := fp.Save(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := fp.Save(fixture(t, idB)); err != nil {
		t.Fatal(err)
	}
	// Strays that must never be treated as sessions.
	for _, name := range []string{".tmp-" + idC + "-123", "notes.json", "0123456789ABCDEF.json", "short.json", idC + ".json.bak", "readme.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, idC+".json"), 0o755); err != nil {
		t.Fatal(err)
	}

	ids, err := fp.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(ids)
	if !reflect.DeepEqual(ids, []string{idA, idB}) {
		t.Fatalf("ListAll = %v", ids)
	}
	// Overwrites leave no temp files behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-"+idA) || strings.HasPrefix(e.Name(), ".tmp-"+idB) {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
	got, _ := fp.Load(idA)
	if got.Seq != 20 {
		t.Fatalf("latest save not visible: seq %d", got.Seq)
	}
}

func TestFileDelete(t *testing.T) {
	fp, dir := newFile(t)
	if err := fp.Save(fixture(t, idA)); err != nil {
		t.Fatal(err)
	}
	if err := fp.Delete(idA); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, idA+".json")); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if err := fp.Delete(idA); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
	if _, err := fp.Load(idA); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load after Delete = %v, want ErrNotFound", err)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	fp, dir := newFile(t)
	parent := filepath.Dir(dir)
	secret := filepath.Join(parent, "secret.json")
	if err := os.WriteFile(secret, []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"../secret", "../../etc/passwd", "..", ".", "", "/abs", `..\secret`, idA + "/../" + idB} {
		bad := fixture(t, idA)
		bad.ID = id
		if err := fp.Save(bad); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Save(%q) = %v, want ErrInvalidID", id, err)
		}
		if _, err := fp.Load(id); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Load(%q) = %v, want ErrInvalidID", id, err)
		}
		if err := fp.Delete(id); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Delete(%q) = %v, want ErrInvalidID", id, err)
		}
		mgr := NewManager(fp, 10)
		if err := mgr.Put(bad); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Put(%q) = %v, want ErrInvalidID", id, err)
		}
		var se *service.Error
		if err := mgr.Delete(id); !errors.As(err, &se) || se.Code != service.CodeNotFound {
			t.Errorf("Manager.Delete(%q) = %v, want NOT_FOUND", id, err)
		}
	}
	if _, err := os.Stat(secret); err != nil {
		t.Fatalf("file outside the sessions dir was touched: %v", err)
	}
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if e.Name() != "sessions" && e.Name() != "secret.json" {
			t.Errorf("unexpected entry outside sessions dir: %s", e.Name())
		}
	}
	if ids, _ := fp.ListAll(); len(ids) != 0 {
		t.Fatalf("rejected ids produced files: %v", ids)
	}
}

// writeFile writes a persisted session after letting mutate edit it.
func writeFile(t *testing.T, dir, id string, mutate func(*PersistedSession)) {
	t.Helper()
	s := fixture(t, id)
	ps := PersistedSession{
		SchemaVersion: SchemaVersion, ID: id, DisplayName: s.DisplayName, Map: s.Map.Config(),
		Program: s.Program, VM: s.VM, SpeedMs: s.SpeedMs, Seq: s.Seq,
		CreatedAt: s.CreatedAt, LastActionAt: s.LastActionAt, Attempts: s.Attempts, BestSteps: s.BestSteps,
	}
	if mutate != nil {
		mutate(&ps)
	}
	data, err := json.Marshal(ps)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsInvalidFiles(t *testing.T) {
	cases := map[string]func(*PersistedSession){
		"future schema":      func(p *PersistedSession) { p.SchemaVersion = 2 },
		"missing schema":     func(p *PersistedSession) { p.SchemaVersion = 0 },
		"id mismatch":        func(p *PersistedSession) { p.ID = idB },
		"map without treats": func(p *PersistedSession) { p.Map.Layout = []string{".....", ">....", "....."} },
		"program wrong size": func(p *PersistedSession) { p.Program.Main = p.Program.Main[:3] },
		"unknown status":     func(p *PersistedSession) { p.VM.Status = "PAUSED" },
		"bad facing":         func(p *PersistedSession) { p.VM.Facing = 9 },
		"position off map":   func(p *PersistedSession) { p.VM.Pos = engine.Position{X: 50, Y: 1} },
		"collected length":   func(p *PersistedSession) { p.VM.Collected = nil },
		"treats_left lies":   func(p *PersistedSession) { p.VM.TreatsLeft = 0 },
		"empty call stack":   func(p *PersistedSession) { p.VM.CallStack = nil },
		"stack tape range":   func(p *PersistedSession) { p.VM.CallStack = append(p.VM.CallStack, engine.Frame{Tape: 5, PC: 0}) },
		"main not first":     func(p *PersistedSession) { p.VM.CallStack[0].Tape = 0 },
		"pc beyond tape":     func(p *PersistedSession) { p.VM.CallStack[0].PC = 99 },
		"steps beyond limit": func(p *PersistedSession) { p.VM.Steps = 10_000 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			fp, dir := newFile(t)
			writeFile(t, dir, idA, mutate)
			if _, err := fp.Load(idA); err == nil {
				t.Fatal("invalid file loaded")
			} else if errors.Is(err, ErrNotFound) {
				t.Fatalf("invalid file reported as not found: %v", err)
			}
		})
	}
}

func TestLoadAllSkipsCorruptFilesAndNormalizesPlaying(t *testing.T) {
	fp, dir := newFile(t)
	writeFile(t, dir, idA, nil)
	writeFile(t, dir, idB, func(p *PersistedSession) { p.SchemaVersion = 99 })
	if err := os.WriteFile(filepath.Join(dir, idC+".json"), []byte("{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0000000000000000.json"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	m := NewManager(fp, 10)
	if err := m.LoadAll(); err != nil {
		t.Fatalf("corrupt files must not be fatal: %v", err)
	}
	if m.Count() != 1 {
		t.Fatalf("loaded %d sessions, want 1", m.Count())
	}
	s, ok := m.Get(idA)
	if !ok || s.Playing {
		t.Fatalf("session A: found=%v playing=%v", ok, s != nil && s.Playing)
	}
}

// stubPersistence returns a Playing session to prove LoadAll normalizes it.
type stubPersistence struct{ s *service.Session }

func (p stubPersistence) Save(*service.Session) error { return nil }
func (p stubPersistence) Load(string) (*service.Session, error) {
	return p.s, nil
}
func (p stubPersistence) Delete(string) error { return ErrNotFound }
func (p stubPersistence) ListAll() ([]string, error) {
	return []string{p.s.ID}, nil
}

func TestLoadAllForcesPlayingFalse(t *testing.T) {
	s := fixture(t, idA)
	s.Playing = true
	m := NewManager(stubPersistence{s}, 10)
	if err := m.LoadAll(); err != nil {
		t.Fatal(err)
	}
	if got, _ := m.Get(idA); got == nil || got.Playing {
		t.Fatal("LoadAll left Playing=true")
	}
}

func TestManagerPutGetListLimit(t *testing.T) {
	m := NewManager(nil, 2)
	a, b, c := fixture(t, idA), fixture(t, idB), fixture(t, idC)
	if err := m.Put(a); err != nil {
		t.Fatal(err)
	}
	if err := m.Put(a); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate Put = %v, want ErrExists", err)
	}
	if err := m.Put(b); err != nil {
		t.Fatal(err)
	}
	err := m.Put(c)
	var se *service.Error
	if !errors.As(err, &se) || se.Code != service.CodeLimit {
		t.Fatalf("third Put = %v, want LIMIT_REACHED", err)
	}
	if got, ok := m.Get(idA); !ok || got != a {
		t.Fatal("Get must return the live pointer")
	}
	if _, ok := m.Get(idC); ok {
		t.Fatal("rejected session was stored")
	}
	if len(m.List()) != 2 || m.Count() != 2 {
		t.Fatalf("List/Count = %d/%d", len(m.List()), m.Count())
	}
	if err := m.Delete(idA); err != nil {
		t.Fatal(err)
	}
	if err := m.Put(c); err != nil {
		t.Fatalf("Put after Delete: %v", err)
	}
	if err := m.Delete(idA); !errors.As(err, &se) || se.Code != service.CodeNotFound {
		t.Fatalf("Delete of missing = %v, want NOT_FOUND", err)
	}
}

func TestManagerDefaultLimit(t *testing.T) {
	if m := NewManager(nil, 0); m.maxSessions != DefaultMaxSessions {
		t.Fatalf("maxSessions = %d, want default %d", m.maxSessions, DefaultMaxSessions)
	}
}

func TestManagerPersistAndDelete(t *testing.T) {
	fp, dir := newFile(t)
	m := NewManager(fp, 10)
	s := fixture(t, idA)
	if err := m.Put(s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, idA+".json")); !os.IsNotExist(err) {
		t.Fatal("Put must not write; the service persists explicitly")
	}
	if err := m.Persist(s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, idA+".json")); err != nil {
		t.Fatalf("Persist wrote nothing: %v", err)
	}
	// Persist snapshots under the session lock.
	s.Lock()
	s.Seq = 99
	s.Unlock()
	if err := m.Persist(s); err != nil {
		t.Fatal(err)
	}
	if got, _ := fp.Load(idA); got.Seq != 99 {
		t.Fatalf("persisted seq = %d, want 99", got.Seq)
	}

	if err := m.Delete(idA); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, idA+".json")); !os.IsNotExist(err) {
		t.Fatal("Delete left the file")
	}
	// A late Persist (e.g. from a runner finishing) must not resurrect it.
	if err := m.Persist(s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, idA+".json")); !os.IsNotExist(err) {
		t.Fatal("Persist resurrected a deleted session")
	}
}

func TestManagerDeleteRemovesFileOfUnloadedSession(t *testing.T) {
	fp, dir := newFile(t)
	writeFile(t, dir, idA, nil)
	m := NewManager(fp, 10) // LoadAll not called: only on disk
	if err := m.Delete(idA); err != nil {
		t.Fatalf("Delete of disk-only session: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, idA+".json")); !os.IsNotExist(err) {
		t.Fatal("file not removed")
	}
}

func TestManagerMemoryOnly(t *testing.T) {
	m := NewManager(nil, 5)
	s := fixture(t, idA)
	if err := m.Put(s); err != nil {
		t.Fatal(err)
	}
	if err := m.Persist(s); err != nil {
		t.Fatalf("Persist without persistence: %v", err)
	}
	if err := m.LoadAll(); err != nil {
		t.Fatalf("LoadAll without persistence: %v", err)
	}
	if err := m.Delete(idA); err != nil {
		t.Fatal(err)
	}
}

func TestManagerLoadAllRoundTripKeepsLimitOnPut(t *testing.T) {
	fp, _ := newFile(t)
	m1 := NewManager(fp, 10)
	for _, id := range []string{idA, idB, idC} {
		s := fixture(t, id)
		if err := m1.Put(s); err != nil {
			t.Fatal(err)
		}
		if err := m1.Persist(s); err != nil {
			t.Fatal(err)
		}
	}
	m2 := NewManager(fp, 2) // fewer slots than persisted sessions
	if err := m2.LoadAll(); err != nil {
		t.Fatal(err)
	}
	if m2.Count() != 3 {
		t.Fatalf("persisted sessions must all load, got %d", m2.Count())
	}
	var se *service.Error
	if err := m2.Put(fixture(t, "00112233445566dd")); !errors.As(err, &se) || se.Code != service.CodeLimit {
		t.Fatalf("Put over limit = %v", err)
	}
}

// TestManagerConcurrentPersist races mutation, Persist and Delete (run with -race).
func TestManagerConcurrentPersist(t *testing.T) {
	fp, _ := newFile(t)
	m := NewManager(fp, 10)
	s := fixture(t, idA)
	if err := m.Put(s); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				s.Lock()
				s.Seq++
				s.Unlock()
				if err := m.Persist(s); err != nil {
					t.Errorf("Persist: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	got, err := fp.Load(idA)
	if err != nil {
		t.Fatal(err)
	}
	s.Lock()
	want := s.Seq
	s.Unlock()
	if got.Seq != want {
		t.Fatalf("last write wins: file has seq %d, live has %d", got.Seq, want)
	}
}
