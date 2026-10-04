package service_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
	"github.com/wricardo/drbrain-motor-programming/game/session"
	"github.com/wricardo/drbrain-motor-programming/transport/websocket"
)

// boot builds a service on top of file persistence in dir, loading what is there.
func boot(t *testing.T, dir string, maps service.MapStore, bc service.Broadcaster) (service.GameService, *session.Manager) {
	t.Helper()
	fp, err := session.NewFilePersistence(dir)
	if err != nil {
		t.Fatal(err)
	}
	mgr := session.NewManager(fp, 100)
	if err := mgr.LoadAll(); err != nil {
		t.Fatal(err)
	}
	svc := service.New(maps, mgr, bc, service.Options{})
	t.Cleanup(svc.Shutdown)
	return svc, mgr
}

func TestRestartRestoresSessionPaused(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	maps := newFakeMaps(t, corridorCfg, spinCfg)

	svc1, _ := boot(t, dir, maps, nil)
	s, err := svc1.CreateSession("spin", "restart me")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc1.SetProgram(s.ID, spinProgram()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc1.Run(s.ID, 50); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "two steps", func() bool {
		got, _ := svc1.GetSession(s.ID)
		return got.VM.Steps >= 2
	})
	// Shut down while Playing: the final state must be persisted with Playing=false.
	svc1.Shutdown()
	want, err := svc1.GetSession(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want.Playing || want.VM.Status != engine.StatusRunning {
		t.Fatalf("pre-restart state: playing=%v status=%s", want.Playing, want.VM.Status)
	}

	raw, err := os.ReadFile(filepath.Join(dir, s.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil || onDisk.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d (%v)", onDisk.SchemaVersion, err)
	}

	// "Restart": brand-new manager and service over the same directory.
	svc2, _ := boot(t, dir, maps, nil)
	got, err := svc2.GetSession(s.ID)
	if err != nil {
		t.Fatalf("session lost across restart: %v", err)
	}
	if got.Playing {
		t.Fatal("restored session is Playing")
	}
	if got.DisplayName != "restart me" || got.MapID != "spin" || got.Seq != want.Seq || got.Attempts != want.Attempts {
		t.Fatalf("metadata differs: %+v", got)
	}
	if !reflect.DeepEqual(got.Program, want.Program) || !reflect.DeepEqual(got.VM, want.VM) {
		t.Fatalf("program/VM not restored intact:\n got %+v\nwant %+v", got.VM, want.VM)
	}
	if service.RunnerCount(svc2) != 0 {
		t.Fatal("a runner started on its own")
	}

	// Run resumes from the saved VM instead of starting over.
	if _, err := svc2.Run(s.ID, 50); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "resumed steps", func() bool {
		cur, _ := svc2.GetSession(s.ID)
		return cur.VM.Steps >= want.VM.Steps+2
	})
	if _, err := svc2.Pause(s.ID); err != nil {
		t.Fatal(err)
	}
}

func TestRestartKeepsOutcomeAndCounters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	maps := newFakeMaps(t, corridorCfg)

	svc1, _ := boot(t, dir, maps, nil)
	s, _ := svc1.CreateSession("corridor", "")
	if _, err := svc1.SetProgram(s.ID, corridorFast()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc1.Run(s.ID, 50); err != nil {
		t.Fatal(err)
	}
	var want *service.Session
	waitFor(t, "win", func() bool {
		want, _ = svc1.GetSession(s.ID)
		return want.VM.Status == engine.StatusWon && !want.Playing
	})
	// The terminal outcome reaches the disk without any Pause/Shutdown (the
	// write follows the in-memory transition, so poll the file).
	fp, err := session.NewFilePersistence(dir)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "terminal outcome on disk", func() bool {
		onDisk, err := fp.Load(s.ID)
		return err == nil && onDisk.VM.Status == engine.StatusWon
	})
	svc2, _ := boot(t, dir, maps, nil)
	got, err := svc2.GetSession(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.VM.Status != engine.StatusWon || got.Attempts != 1 || got.BestSteps != 4 || !reflect.DeepEqual(got.VM, want.VM) {
		t.Fatalf("restored: %s attempts %d best %d", got.VM.Status, got.Attempts, got.BestSteps)
	}
	if _, err := svc2.Run(s.ID, 50); err == nil {
		t.Fatal("Run on a restored terminal session succeeded")
	} else {
		wantCode(t, err, service.CodeSessionTerminal)
	}
}

func TestDeleteSessionRemovesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	maps := newFakeMaps(t, spinCfg)
	svc, _ := boot(t, dir, maps, nil)
	s, _ := svc.CreateSession("spin", "")
	if _, err := svc.SetProgram(s.ID, spinProgram()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Run(s.ID, 50); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, s.ID+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file missing: %v", err)
	}
	if err := svc.DeleteSession(s.ID); err != nil {
		t.Fatal(err)
	}
	svc.Shutdown() // persists everything still registered: must not bring the file back
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("deleted session file exists: %v", err)
	}
	svc2, _ := boot(t, dir, maps, nil)
	if _, err := svc2.GetSession(s.ID); err == nil {
		t.Fatal("deleted session came back after restart")
	}
}

func TestMaxSessionsFromManager(t *testing.T) {
	maps := newFakeMaps(t, corridorCfg)
	mgr := session.NewManager(nil, 2)
	svc := service.New(maps, mgr, nil, service.Options{})
	t.Cleanup(svc.Shutdown)
	for range 2 {
		if _, err := svc.CreateSession("corridor", ""); err != nil {
			t.Fatal(err)
		}
	}
	_, err := svc.CreateSession("corridor", "")
	wantCode(t, err, service.CodeLimit)
}

// TestSlowSubscriberStillGetsTerminalUpdate runs a 19-step win (more than the
// 8-slot subscriber buffer) with a subscriber that reads nothing until the run
// has finished: the final WON snapshot must still be delivered.
func TestSlowSubscriberStillGetsTerminalUpdate(t *testing.T) {
	hub := websocket.NewHub()
	e := newEnv(t, envOpts{bc: hub})
	s := e.create(t, "long")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := hub.SubscribeSession(ctx, s.ID)

	e.setProgram(t, s.ID, longWin())
	e.run(t, s.ID, 50)
	done := e.waitTerminal(t, s.ID)
	if done.VM.Status != engine.StatusWon || done.VM.Steps != 19 {
		t.Fatalf("run ended %s in %d steps", done.VM.Status, done.VM.Steps)
	}

	var got []*service.Session
	for len(updates) > 0 {
		got = append(got, <-updates)
	}
	if len(got) == 0 || len(got) > 8 {
		t.Fatalf("buffered %d updates, want 1..8", len(got))
	}
	last := got[len(got)-1]
	if last.VM.Status != engine.StatusWon || last.Playing || last.Seq != done.Seq {
		t.Fatalf("last buffered update is %s playing=%v seq %d, want WON seq %d", last.VM.Status, last.Playing, last.Seq, done.Seq)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Seq <= got[i-1].Seq {
			t.Fatalf("seq not strictly increasing: %d then %d", got[i-1].Seq, got[i].Seq)
		}
	}
}

// TestSubscriberSeesPauseAndReset: spectators must see control mutations too.
func TestSubscriberSeesPauseAndReset(t *testing.T) {
	hub := websocket.NewHub()
	e := newEnv(t, envOpts{bc: hub})
	s := e.create(t, "spin")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := hub.SubscribeSession(ctx, s.ID)

	next := func() *service.Session {
		t.Helper()
		select {
		case u := <-updates:
			return u
		case <-ctx.Done():
			t.Fatal("cancelled")
			return nil
		}
	}

	e.setProgram(t, s.ID, spinProgram())
	if u := next(); u.Playing || u.Program.Main[0] != engine.CallSub1 {
		t.Fatalf("SetProgram update: %+v", u)
	}
	e.run(t, s.ID, 50)
	if u := next(); !u.Playing {
		t.Fatal("Run update not playing")
	}
	var u *service.Session
	for u = next(); u.VM.Steps < 1; u = next() {
	}
	if _, err := e.svc.Pause(s.ID); err != nil {
		t.Fatal(err)
	}
	for u = next(); u.Playing; u = next() {
	}
	if _, err := e.svc.Reset(s.ID); err != nil {
		t.Fatal(err)
	}
	if u = next(); u.VM.Steps != 0 || u.VM.Status != engine.StatusReady {
		t.Fatalf("Reset update: steps %d status %s", u.VM.Steps, u.VM.Status)
	}
}
