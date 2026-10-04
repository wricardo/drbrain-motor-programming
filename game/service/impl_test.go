package service_test

import (
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
	"github.com/wricardo/drbrain-motor-programming/game/session"
)

const tick = 50 * time.Millisecond // fastest allowed speed

func TestCreateSession(t *testing.T) {
	e := newEnv(t, envOpts{})

	s, err := e.svc.CreateSession("corridor", "  Ada  ")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(s.ID) {
		t.Fatalf("id %q is not 8 random bytes hex", s.ID)
	}
	if s.DisplayName != "Ada" || s.MapID != "corridor" {
		t.Fatalf("name/map = %q/%q", s.DisplayName, s.MapID)
	}
	if s.Seq != 1 || s.Playing || s.Attempts != 0 || s.BestSteps != 0 || s.SpeedMs != service.DefaultSpeedMs {
		t.Fatalf("unexpected initial state: %+v", s)
	}
	if s.VM.Status != engine.StatusReady || s.VM.Steps != 0 {
		t.Fatalf("VM = %s steps %d, want fresh READY", s.VM.Status, s.VM.Steps)
	}
	if !reflect.DeepEqual(s.Program, engine.EmptyProgram(s.Map)) {
		t.Fatalf("program is not the all-EMPTY program: %+v", s.Program)
	}
	if e.pers.saveCount(s.ID) != 1 {
		t.Fatalf("saves = %d, want 1 on create", e.pers.saveCount(s.ID))
	}

	// A second session gets a different id.
	if s2 := e.create(t, "corridor"); s2.ID == s.ID {
		t.Fatal("duplicate session id")
	}

	_, err = e.svc.CreateSession("nope", "")
	wantCode(t, err, service.CodeNotFound)
	_, err = e.svc.CreateSession("corridor", strings.Repeat("x", 41))
	wantCode(t, err, service.CodeInvalidArgument)
	if _, err = e.svc.CreateSession("corridor", strings.Repeat("é", 40)); err != nil {
		t.Fatalf("40 multibyte characters must fit: %v", err)
	}
}

func TestCustomDisplayNameLimit(t *testing.T) {
	e := newEnv(t, envOpts{svcOpts: service.Options{MaxDisplayName: 5}})
	if _, err := e.svc.CreateSession("corridor", "abcdef"); err == nil {
		t.Fatal("6 characters accepted with limit 5")
	} else {
		wantCode(t, err, service.CodeInvalidArgument)
	}
	if _, err := e.svc.CreateSession("corridor", " abcde "); err != nil {
		t.Fatalf("name is limited after trimming: %v", err)
	}
}

func TestRenameSession(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	r, err := e.svc.RenameSession(s.ID, "  Grace ")
	if err != nil {
		t.Fatal(err)
	}
	if r.DisplayName != "Grace" || r.Seq != s.Seq+1 {
		t.Fatalf("rename: name %q seq %d (was %d)", r.DisplayName, r.Seq, s.Seq)
	}
	if e.pers.saveCount(s.ID) != 2 {
		t.Fatalf("saves = %d, want create+rename", e.pers.saveCount(s.ID))
	}
	last := e.rec.forSession(s.ID)
	if got := last[len(last)-1]; got.DisplayName != "Grace" {
		t.Fatalf("rename not broadcast: %q", got.DisplayName)
	}
	_, err = e.svc.RenameSession(s.ID, strings.Repeat("x", 41))
	wantCode(t, err, service.CodeInvalidArgument)
	_, err = e.svc.RenameSession("0000000000000000", "x")
	wantCode(t, err, service.CodeNotFound)
	if got := e.get(t, s.ID); got.DisplayName != "Grace" {
		t.Fatalf("failed rename changed name to %q", got.DisplayName)
	}
}

func TestSnapshotsAreIsolatedFromLiveSession(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.setProgram(t, s.ID, corridorFast())

	snap := e.get(t, s.ID)
	snap.Program.Main[0] = engine.TurnLeft
	snap.VM.Visited[0] = engine.Position{X: 99, Y: 99}
	snap.VM.CallStack[0].PC = 42

	again := e.get(t, s.ID)
	if again.Program.Main[0] != engine.MoveForward || again.VM.Visited[0] == (engine.Position{X: 99, Y: 99}) || again.VM.CallStack[0].PC == 42 {
		t.Fatal("mutating a returned snapshot changed the live session")
	}
}

func TestMapSnapshotIsolation(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	storeMap, _ := e.maps.Get("corridor")
	if s.Map == storeMap {
		t.Fatal("session shares the map store's *Map instead of a snapshot")
	}
	e.setProgram(t, s.ID, corridorFast())

	// Replace the map under the same id with a layout where the program cannot win,
	// then delete it entirely.
	edited := corridorCfg
	edited.Layout = []string{".....", ">....", "...*."}
	if _, err := e.svc.SaveMap(edited); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteMap("corridor"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.GetMap("corridor"); err == nil {
		t.Fatal("map still present")
	}

	got := e.get(t, s.ID)
	if !reflect.DeepEqual(got.Map.Layout, corridorCfg.Layout) {
		t.Fatalf("snapshot layout changed: %v", got.Map.Layout)
	}
	e.run(t, s.ID, 50)
	done := e.waitTerminal(t, s.ID)
	if done.VM.Status != engine.StatusWon || done.VM.Steps != 4 {
		t.Fatalf("after map edit/delete: %s in %d steps, want WON in 4", done.VM.Status, done.VM.Steps)
	}
	_, err := e.svc.CreateSession("corridor", "")
	wantCode(t, err, service.CodeNotFound)
}

func TestSetProgramValidationAndReset(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	bad := map[string]engine.Program{
		"short main":     {Main: pad(5), Subs: [][]engine.Instruction{pad(2)}},
		"missing subs":   {Main: pad(6)},
		"wrong sub size": {Main: pad(6), Subs: [][]engine.Instruction{pad(3)}},
		"unknown ins":    {Main: pad(6, "JUMP"), Subs: [][]engine.Instruction{pad(2)}},
		"sub 2 absent":   {Main: pad(6, engine.CallSub2), Subs: [][]engine.Instruction{pad(2)}},
	}
	for name, p := range bad {
		_, err := e.svc.SetProgram(s.ID, p)
		if err == nil {
			t.Fatalf("%s: accepted", name)
		}
		wantCode(t, err, service.CodeInvalidProgram)
	}
	if got := e.get(t, s.ID); got.Seq != s.Seq || !reflect.DeepEqual(got.Program, s.Program) {
		t.Fatal("rejected program changed the session")
	}
	_, err := e.svc.SetProgram("0000000000000000", corridorFast())
	wantCode(t, err, service.CodeNotFound)

	// Stored program is a copy; VM is reset and settled past leading EMPTYs.
	p := engine.Program{Main: pad(6, engine.Empty, engine.Empty, engine.MoveForward), Subs: [][]engine.Instruction{pad(2)}}
	got := e.setProgram(t, s.ID, p)
	p.Main[2] = engine.TurnLeft
	if e.get(t, s.ID).Program.Main[2] != engine.MoveForward {
		t.Fatal("SetProgram kept the caller's slice")
	}
	if got.Seq != s.Seq+1 || got.VM.Status != engine.StatusReady || got.VM.Steps != 0 {
		t.Fatalf("after SetProgram: seq %d status %s steps %d", got.Seq, got.VM.Status, got.VM.Steps)
	}
	if top := got.VM.CallStack[len(got.VM.CallStack)-1]; top.PC != 2 {
		t.Fatalf("VM not settled: top frame pc = %d, want 2", top.PC)
	}
}

func TestRunToWonStopsRunner(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.setProgram(t, s.ID, corridorFast())

	r, err := e.svc.Run(s.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Playing || r.SpeedMs != 50 {
		t.Fatalf("Run snapshot: playing=%v speed=%d", r.Playing, r.SpeedMs)
	}
	done := e.waitTerminal(t, s.ID)

	if done.VM.Status != engine.StatusWon || done.VM.Steps != 4 || done.VM.TreatsLeft != 0 {
		t.Fatalf("result: %s steps %d treats left %d", done.VM.Status, done.VM.Steps, done.VM.TreatsLeft)
	}
	if done.Attempts != 1 || done.BestSteps != 4 {
		t.Fatalf("attempts/best = %d/%d, want 1/4", done.Attempts, done.BestSteps)
	}
	if done.LastEvent == nil || done.LastEvent.Status != engine.StatusWon || done.LastEvent.Step != 4 {
		t.Fatalf("LastEvent = %+v", done.LastEvent)
	}
	waitFor(t, "runner to deregister", func() bool { return service.RunnerCount(e.svc) == 0 })
	// persisted: create, setProgram, terminal outcome (nothing per tick). The
	// terminal write follows the in-memory transition, so wait for it.
	waitFor(t, "terminal persist", func() bool { return e.pers.saveCount(s.ID) >= 3 })
	if n := e.pers.saveCount(s.ID); n != 3 {
		t.Fatalf("saves = %d, want 3 (create, setProgram, terminal)", n)
	}
	if saved := e.pers.latest(s.ID); saved.VM.Status != engine.StatusWon || saved.Attempts != 1 {
		t.Fatalf("persisted state is stale: %s attempts %d", saved.VM.Status, saved.Attempts)
	}
	// No more ticks after the terminal state.
	seq := done.Seq
	time.Sleep(3 * tick)
	if got := e.get(t, s.ID); got.Seq != seq {
		t.Fatalf("seq moved after terminal: %d -> %d", seq, got.Seq)
	}
}

func TestBroadcastSeqMonotonicAndComplete(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.setProgram(t, s.ID, corridorFast())
	e.run(t, s.ID, 50)
	done := e.waitTerminal(t, s.ID)
	waitFor(t, "final broadcast", func() bool {
		snaps := e.rec.forSession(s.ID)
		return snaps[len(snaps)-1].Seq == done.Seq
	})

	snaps := e.rec.forSession(s.ID)
	// create, setProgram, run, 4 ticks.
	if len(snaps) != 7 {
		t.Fatalf("%d broadcasts, want 7", len(snaps))
	}
	for i, sn := range snaps {
		if sn.Seq != uint64(i+1) {
			t.Fatalf("broadcast %d has seq %d, want %d (every change +1, in order)", i, sn.Seq, i+1)
		}
	}
	if last := snaps[len(snaps)-1]; last.VM.Status != engine.StatusWon || last.Playing {
		t.Fatalf("final broadcast: %s playing=%v", last.VM.Status, last.Playing)
	}
	if !snaps[2].Playing || snaps[3].LastEvent == nil || snaps[3].LastEvent.Step != 1 {
		t.Fatal("run/tick snapshots missing playing flag or last event")
	}
}

func TestPauseStopsTickingAndPersists(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "spin")
	e.setProgram(t, s.ID, spinProgram())
	e.run(t, s.ID, 50)
	e.waitSteps(t, s.ID, 2)
	if n := e.pers.saveCount(s.ID); n != 2 {
		t.Fatalf("saves while running = %d, want 2 (create, setProgram; ticks never persist)", n)
	}

	p, err := e.svc.Pause(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Playing || p.VM.Status != engine.StatusRunning {
		t.Fatalf("paused: playing=%v status=%s", p.Playing, p.VM.Status)
	}
	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("runner still registered after Pause")
	}
	if n := e.pers.saveCount(s.ID); n != 3 {
		t.Fatalf("saves after pause = %d, want 3", n)
	}
	time.Sleep(4 * tick)
	if got := e.get(t, s.ID); got.VM.Steps != p.VM.Steps || got.Seq != p.Seq {
		t.Fatalf("session advanced after Pause: steps %d->%d seq %d->%d", p.VM.Steps, got.VM.Steps, p.Seq, got.Seq)
	}

	// Pause is idempotent and leaves state alone.
	again, err := e.svc.Pause(s.ID)
	if err != nil || again.Seq != p.Seq {
		t.Fatalf("second Pause: %v seq %d want %d", err, again.Seq, p.Seq)
	}

	// Run resumes from the saved VM instead of restarting.
	e.run(t, s.ID, 50)
	e.waitSteps(t, s.ID, p.VM.Steps+2)
	if _, err := e.svc.Pause(s.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSetProgramStepRunRejectedWhilePlaying(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "spin")
	e.setProgram(t, s.ID, spinProgram())
	e.run(t, s.ID, 50)

	_, err := e.svc.SetProgram(s.ID, spinProgram())
	wantCode(t, err, service.CodeSessionPlaying)
	_, err = e.svc.Run(s.ID, 50)
	wantCode(t, err, service.CodeSessionPlaying)
	_, err = e.svc.Step(s.ID)
	wantCode(t, err, service.CodeSessionPlaying)

	if _, err := e.svc.Pause(s.ID); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.SetProgram(s.ID, spinProgram())
	if err != nil {
		t.Fatalf("SetProgram after Pause: %v", err)
	}
	if got.VM.Steps != 0 || got.VM.Status != engine.StatusReady {
		t.Fatalf("SetProgram did not reset the VM: steps %d status %s", got.VM.Steps, got.VM.Status)
	}
}

func TestRunValidation(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "spin")
	e.setProgram(t, s.ID, spinProgram())

	for _, speed := range []int{-1, 0, 49, 5001} {
		_, err := e.svc.Run(s.ID, speed)
		wantCode(t, err, service.CodeInvalidArgument)
	}
	_, err := e.svc.Run("0000000000000000", 100)
	wantCode(t, err, service.CodeNotFound)
	if got := e.get(t, s.ID); got.Playing || service.RunnerCount(e.svc) != 0 {
		t.Fatal("rejected Run started a runner")
	}
	for _, speed := range []int{service.MinSpeedMs, service.MaxSpeedMs} {
		e.run(t, s.ID, speed)
		if got, err := e.svc.Pause(s.ID); err != nil || got.SpeedMs != speed {
			t.Fatalf("speed %d: err=%v got %d", speed, err, got.SpeedMs)
		}
	}
}

func TestTerminalSessionRejectsRunAndStepUntilReset(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.setProgram(t, s.ID, corridorFast())
	e.run(t, s.ID, 50)
	e.waitTerminal(t, s.ID)

	_, err := e.svc.Run(s.ID, 50)
	wantCode(t, err, service.CodeSessionTerminal)
	_, err = e.svc.Step(s.ID)
	wantCode(t, err, service.CodeSessionTerminal)
	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("rejected Run left a runner")
	}

	r, err := e.svc.Reset(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.VM.Status != engine.StatusReady || r.VM.Steps != 0 || r.LastEvent != nil || r.Attempts != 1 || r.BestSteps != 4 {
		t.Fatalf("Reset: status %s steps %d event %v attempts %d best %d", r.VM.Status, r.VM.Steps, r.LastEvent, r.Attempts, r.BestSteps)
	}
	if !reflect.DeepEqual(r.Program, corridorFast()) {
		t.Fatal("Reset dropped the program")
	}
	e.run(t, s.ID, 50)
	if again := e.waitTerminal(t, s.ID); again.Attempts != 2 || again.VM.Status != engine.StatusWon {
		t.Fatalf("replay after reset: attempts %d status %s", again.Attempts, again.VM.Status)
	}
}

func TestManualStep(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.setProgram(t, s.ID, corridorFast())

	prev := e.get(t, s.ID)
	for i := 1; i <= 4; i++ {
		got, err := e.svc.Step(s.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if got.Playing || got.VM.Steps != i || got.Seq != prev.Seq+1 {
			t.Fatalf("step %d: playing=%v steps=%d seq=%d (prev %d)", i, got.Playing, got.VM.Steps, got.Seq, prev.Seq)
		}
		if got.LastEvent == nil || got.LastEvent.Step != i || got.LastEvent.Instruction != engine.MoveForward {
			t.Fatalf("step %d: LastEvent %+v", i, got.LastEvent)
		}
		wantStatus := engine.StatusRunning
		if i == 4 {
			wantStatus = engine.StatusWon
		}
		if got.VM.Status != wantStatus {
			t.Fatalf("step %d: status %s want %s", i, got.VM.Status, wantStatus)
		}
		if n := e.pers.saveCount(s.ID); (i < 4 && n != 2) || (i == 4 && n != 3) {
			t.Fatalf("step %d: saves = %d (persist only on the terminal outcome)", i, n)
		}
		prev = got
	}
	if prev.Attempts != 1 || prev.BestSteps != 4 {
		t.Fatalf("attempts/best = %d/%d", prev.Attempts, prev.BestSteps)
	}
	_, err := e.svc.Step(s.ID)
	wantCode(t, err, service.CodeSessionTerminal)
	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("manual stepping created a runner")
	}
}

func TestAttemptsAndBestSteps(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")

	play := func(p engine.Program) *service.Session {
		t.Helper()
		e.setProgram(t, s.ID, p)
		e.run(t, s.ID, 50)
		return e.waitTerminal(t, s.ID)
	}

	if got := play(corridorSlow()); got.VM.Steps != 6 || got.Attempts != 1 || got.BestSteps != 6 {
		t.Fatalf("slow win: steps %d attempts %d best %d", got.VM.Steps, got.Attempts, got.BestSteps)
	}
	// SetProgram and Reset never count as attempts.
	e.setProgram(t, s.ID, corridorFast())
	if got, _ := e.svc.Reset(s.ID); got.Attempts != 1 || got.BestSteps != 6 {
		t.Fatalf("SetProgram/Reset changed counters: %d/%d", got.Attempts, got.BestSteps)
	}
	if got := play(corridorFast()); got.Attempts != 2 || got.BestSteps != 4 {
		t.Fatalf("faster win: attempts %d best %d, want 2/4", got.Attempts, got.BestSteps)
	}
	if got := play(corridorSlow()); got.Attempts != 3 || got.BestSteps != 4 {
		t.Fatalf("slower win must not raise best: attempts %d best %d", got.Attempts, got.BestSteps)
	}
	got := play(corridorLose())
	if got.VM.Status != engine.StatusLost || got.VM.LossReason != engine.LossProgramEnded || got.Attempts != 4 || got.BestSteps != 4 {
		t.Fatalf("loss: %s/%s attempts %d best %d", got.VM.Status, got.VM.LossReason, got.Attempts, got.BestSteps)
	}
}

func TestEmptyProgramLosesWhenRun(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "corridor")
	e.setProgram(t, s.ID, engine.EmptyProgram(s.Map))
	if got := e.get(t, s.ID); got.VM.Status != engine.StatusReady {
		t.Fatalf("an unrun empty program must stay READY, got %s", got.VM.Status)
	}
	r, err := e.svc.Run(s.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if r.Playing || r.VM.Status != engine.StatusLost || r.VM.LossReason != engine.LossProgramEnded || r.Attempts != 1 || r.VM.Steps != 0 {
		t.Fatalf("empty run: playing=%v %s/%s attempts %d steps %d", r.Playing, r.VM.Status, r.VM.LossReason, r.Attempts, r.VM.Steps)
	}
	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("runner left behind")
	}
	// Same through manual Step on a fresh empty session.
	s2 := e.create(t, "corridor")
	st, err := e.svc.Step(s2.ID)
	if err != nil || st.VM.Status != engine.StatusLost || st.Attempts != 1 {
		t.Fatalf("empty Step: %v %+v", err, st)
	}
}

func TestDeleteMidRun(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "spin")
	e.setProgram(t, s.ID, spinProgram())
	e.run(t, s.ID, 50)
	e.waitSteps(t, s.ID, 2)

	if err := e.svc.DeleteSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("runner registered after DeleteSession")
	}
	_, err := e.svc.GetSession(s.ID)
	wantCode(t, err, service.CodeNotFound)
	wantCode(t, e.svc.DeleteSession(s.ID), service.CodeNotFound)

	n := e.rec.count()
	time.Sleep(4 * tick)
	if e.rec.count() != n {
		t.Fatal("broadcasts continued after DeleteSession")
	}
	if _, err := e.svc.Run(s.ID, 50); err == nil {
		t.Fatal("Run on deleted session succeeded")
	}
	list, _ := e.svc.ListSessions(service.SessionFilter{})
	if len(list) != 0 {
		t.Fatalf("deleted session still listed: %d", len(list))
	}
}

func TestResetMidRun(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "spin")
	e.setProgram(t, s.ID, spinProgram())
	e.run(t, s.ID, 50)
	e.waitSteps(t, s.ID, 2)

	r, err := e.svc.Reset(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Playing || r.VM.Status != engine.StatusReady || r.VM.Steps != 0 || r.Attempts != 0 {
		t.Fatalf("Reset mid-run: playing=%v %s steps %d attempts %d", r.Playing, r.VM.Status, r.VM.Steps, r.Attempts)
	}
	if !reflect.DeepEqual(r.Program, spinProgram()) {
		t.Fatal("program lost")
	}
	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("runner still registered")
	}
	time.Sleep(4 * tick)
	if got := e.get(t, s.ID); got.Seq != r.Seq || got.VM.Steps != 0 {
		t.Fatalf("session moved after Reset: seq %d->%d steps %d", r.Seq, got.Seq, got.VM.Steps)
	}
	// And it can be run again from scratch.
	e.run(t, s.ID, 50)
	e.waitSteps(t, s.ID, 1)
	if _, err := e.svc.Pause(s.ID); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownStopsRunnersAndPersists(t *testing.T) {
	e := newEnv(t, envOpts{})
	var ids []string
	for range 3 {
		s := e.create(t, "spin")
		e.setProgram(t, s.ID, spinProgram())
		e.run(t, s.ID, 50)
		ids = append(ids, s.ID)
	}
	for _, id := range ids {
		e.waitSteps(t, id, 2)
	}
	e.svc.Shutdown()

	if service.RunnerCount(e.svc) != 0 {
		t.Fatal("runners left after Shutdown")
	}
	for _, id := range ids {
		got := e.get(t, id)
		if got.Playing {
			t.Fatalf("%s still Playing after Shutdown", id)
		}
		saved := e.pers.latest(id)
		if saved == nil || saved.VM.Steps != got.VM.Steps || saved.Seq != got.Seq {
			t.Fatalf("%s: persisted state is not the final state", id)
		}
	}
	steps := e.get(t, ids[0]).VM.Steps
	time.Sleep(3 * tick)
	if e.get(t, ids[0]).VM.Steps != steps {
		t.Fatal("session advanced after Shutdown")
	}
	if _, err := e.svc.Run(ids[0], 50); err == nil {
		t.Fatal("Run after Shutdown succeeded")
	}
	e.svc.Shutdown() // idempotent
}

func TestSimulate(t *testing.T) {
	e := newEnv(t, envOpts{})

	res, err := e.svc.Simulate("corridor", corridorFast(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != engine.StatusWon || res.Steps != 4 || len(res.Events) != 4 || res.Map == nil || res.FinalState.Steps != 4 {
		t.Fatalf("simulate: %s steps %d events %d", res.Status, res.Steps, len(res.Events))
	}
	res, err = e.svc.Simulate("corridor", corridorFast(), false)
	if err != nil || res.Status != engine.StatusWon || res.Steps != 4 || res.Events != nil {
		t.Fatalf("no-events simulate: %v %+v", err, res)
	}

	// Events are bounded by the map's max_steps.
	res, err = e.svc.Simulate("spin", spinProgram(), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != engine.StatusLost || res.LossReason != engine.LossStepLimit || res.Steps != 100 || len(res.Events) != 100 {
		t.Fatalf("spin: %s/%s steps %d events %d", res.Status, res.LossReason, res.Steps, len(res.Events))
	}

	corridor, _ := e.maps.Get("corridor")
	res, err = e.svc.Simulate("corridor", engine.EmptyProgram(corridor), false)
	if err != nil || res.Status != engine.StatusLost || res.LossReason != engine.LossProgramEnded || res.Steps != 0 {
		t.Fatalf("empty program: %v %+v", err, res)
	}

	_, err = e.svc.Simulate("corridor", engine.Program{Main: pad(2)}, false)
	wantCode(t, err, service.CodeInvalidProgram)
	_, err = e.svc.Simulate("nope", corridorFast(), false)
	wantCode(t, err, service.CodeNotFound)
	if e.mgr.Count() != 0 {
		t.Fatal("Simulate created a session")
	}
}

func TestZeroSubMapSession(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "nosubs")
	if len(s.Map.SubTapeLengths) != 0 || len(s.Program.Subs) != 0 {
		t.Fatalf("snapshot of a zero-sub map grew subs: %v / %d program subs", s.Map.SubTapeLengths, len(s.Program.Subs))
	}
	e.setProgram(t, s.ID, engine.Program{Main: moves(4), Subs: [][]engine.Instruction{}})
	e.run(t, s.ID, 50)
	if got := e.waitTerminal(t, s.ID); got.VM.Status != engine.StatusWon {
		t.Fatalf("status %s", got.VM.Status)
	}
}

func TestListSessions(t *testing.T) {
	e := newEnv(t, envOpts{})
	a := e.create(t, "corridor")
	time.Sleep(3 * time.Millisecond)
	b := e.create(t, "long")
	time.Sleep(3 * time.Millisecond)
	c := e.create(t, "corridor")
	time.Sleep(3 * time.Millisecond)
	// Touch a so it is the most recently active.
	if _, err := e.svc.RenameSession(a.ID, "a"); err != nil {
		t.Fatal(err)
	}

	ids := func(f service.SessionFilter) []string {
		t.Helper()
		l, err := e.svc.ListSessions(f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, s := range l {
			out = append(out, s.ID)
		}
		return out
	}
	if got, want := ids(service.SessionFilter{}), []string{a.ID, c.ID, b.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recent = %v, want %v", got, want)
	}
	if got, want := ids(service.SessionFilter{Sort: "created"}), []string{c.ID, b.ID, a.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("created = %v, want %v", got, want)
	}
	if got, want := ids(service.SessionFilter{Sort: "CREATED", Limit: 2}), []string{c.ID, b.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("limit 2 = %v, want %v", got, want)
	}
	if got, want := ids(service.SessionFilter{MapID: "corridor"}), []string{a.ID, c.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("map filter = %v, want %v", got, want)
	}
	if got := ids(service.SessionFilter{MapID: "unknown"}); len(got) != 0 {
		t.Fatalf("unknown map filter = %v", got)
	}
	if got := ids(service.SessionFilter{Limit: 100000}); len(got) != 3 {
		t.Fatalf("limit is clamped, not rejected: %v", got)
	}
	_, err := e.svc.ListSessions(service.SessionFilter{Limit: -1})
	wantCode(t, err, service.CodeInvalidArgument)
	_, err = e.svc.ListSessions(service.SessionFilter{Sort: "bogus"})
	wantCode(t, err, service.CodeInvalidArgument)
}

func TestListSessionsDefaultLimit(t *testing.T) {
	e := newEnv(t, envOpts{maxSessions: 600})
	for range service.MaxListLimit + 1 {
		e.create(t, "corridor")
	}
	l, _ := e.svc.ListSessions(service.SessionFilter{})
	if len(l) != service.DefaultListLimit {
		t.Fatalf("default limit: %d, want %d", len(l), service.DefaultListLimit)
	}
	l, _ = e.svc.ListSessions(service.SessionFilter{Limit: 100000})
	if len(l) != service.MaxListLimit {
		t.Fatalf("max limit: %d, want %d", len(l), service.MaxListLimit)
	}
}

func TestMaxSessions(t *testing.T) {
	e := newEnv(t, envOpts{maxSessions: 2})
	a := e.create(t, "corridor")
	e.create(t, "corridor")
	_, err := e.svc.CreateSession("corridor", "")
	wantCode(t, err, service.CodeLimit)
	if err := e.svc.DeleteSession(a.ID); err != nil {
		t.Fatal(err)
	}
	e.create(t, "corridor")
}

func TestSessionTTLSweptOnNew(t *testing.T) {
	maps := newFakeMaps(t, corridorCfg)
	mgr := session.NewManager(nil, 10)
	m := mustMap(t, corridorCfg)
	prog := engine.EmptyProgram(m)
	put := func(id string, idle time.Duration) {
		now := time.Now()
		s := &service.Session{
			ID: id, MapID: m.ID, Map: m, Program: prog, VM: engine.NewVMState(m), SpeedMs: 500,
			CreatedAt: now.Add(-idle), LastActionAt: now.Add(-idle),
		}
		if err := mgr.Put(s); err != nil {
			t.Fatal(err)
		}
	}
	put("0000000000000001", 48*time.Hour)
	put("0000000000000002", time.Hour)

	svc := service.New(maps, mgr, nil, service.Options{SessionTTL: 24 * time.Hour})
	t.Cleanup(svc.Shutdown)
	if _, ok := mgr.Get("0000000000000001"); ok {
		t.Fatal("expired session survived")
	}
	if _, ok := mgr.Get("0000000000000002"); !ok {
		t.Fatal("fresh session was swept")
	}
}

func TestNoGoroutineLeak(t *testing.T) {
	base := runtime.NumGoroutine()
	e := newEnv(t, envOpts{})

	var ids []string
	for range 6 {
		s := e.create(t, "spin")
		e.setProgram(t, s.ID, spinProgram())
		e.run(t, s.ID, 50)
		ids = append(ids, s.ID)
	}
	w := e.create(t, "corridor")
	e.setProgram(t, w.ID, corridorFast())
	e.run(t, w.ID, 50)
	e.waitTerminal(t, w.ID)

	if _, err := e.svc.Pause(ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Reset(ids[1]); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteSession(ids[2]); err != nil {
		t.Fatal(err)
	}
	e.svc.Shutdown()

	waitFor(t, "goroutines to exit", func() bool { return runtime.NumGoroutine() <= base })
}

// TestConcurrentControls hammers one session from many goroutines while a
// runner ticks and finally deletes it mid-flight; it must be race-free, only
// return coded errors and leave no runner behind.
func TestConcurrentControls(t *testing.T) {
	e := newEnv(t, envOpts{})
	s := e.create(t, "spin")
	e.setProgram(t, s.ID, spinProgram())

	allowed := map[string]bool{
		service.CodeSessionPlaying: true, service.CodeSessionTerminal: true, service.CodeNotFound: true,
	}
	check := func(err error) bool {
		if err == nil {
			return true
		}
		if se, ok := err.(*service.Error); ok && allowed[se.Code] {
			return true
		}
		t.Errorf("unexpected error: %v", err)
		return false
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	// A "player" lets the runner tick for a while between Run and Pause.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			_, err := e.svc.Run(s.ID, 50)
			if !check(err) {
				return
			}
			time.Sleep(70 * time.Millisecond)
			_, err = e.svc.Pause(s.ID)
			if !check(err) {
				return
			}
		}
	}()
	for g := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-done:
					return
				default:
				}
				var err error
				switch (i + g) % 8 {
				case 0:
					_, err = e.svc.Run(s.ID, 50)
				case 1:
					_, err = e.svc.Pause(s.ID)
				case 2:
					_, err = e.svc.Step(s.ID)
				case 3:
					_, err = e.svc.Reset(s.ID)
				case 4:
					_, err = e.svc.SetProgram(s.ID, spinProgram())
				case 5:
					_, err = e.svc.RenameSession(s.ID, "n")
				case 6:
					_, err = e.svc.GetSession(s.ID)
				case 7:
					_, err = e.svc.ListSessions(service.SessionFilter{})
				}
				if !check(err) {
					return
				}
				runtime.Gosched()
			}
		}()
	}

	time.Sleep(400 * time.Millisecond)
	if err := e.svc.DeleteSession(s.ID); err != nil {
		t.Errorf("delete: %v", err)
	}
	close(done)
	wg.Wait()

	if n := service.RunnerCount(e.svc); n != 0 {
		t.Fatalf("%d runners left after the session was deleted", n)
	}
}
