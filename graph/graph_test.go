package graph

import (
	"context"
	"errors"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
	"github.com/wricardo/drbrain-motor-programming/graph/model"
	"github.com/wricardo/drbrain-motor-programming/transport/websocket"
)

// fakeService overrides only what the tests exercise; any other call panics
// on the nil embedded interface, which flags an unexpected dependency.
type fakeService struct {
	service.GameService
	maps     map[string]*engine.Map
	sessions map[string]*service.Session
	saved    []engine.MapConfig
	err      error
}

func (f *fakeService) GetMap(id string) (*engine.Map, error) {
	if f.err != nil {
		return nil, f.err
	}
	if m, ok := f.maps[id]; ok {
		return m, nil
	}
	return nil, service.Errorf(service.CodeNotFound, "map %q not found", id)
}

func (f *fakeService) SaveMap(cfg engine.MapConfig) (*engine.Map, error) {
	m, err := engine.NewMap(cfg)
	if err != nil {
		return nil, err
	}
	f.saved = append(f.saved, cfg)
	return m, nil
}

func (f *fakeService) CreateMap(cfg engine.MapConfig) (*engine.Map, error) {
	if f.err != nil {
		return nil, f.err
	}
	if _, ok := f.maps[cfg.ID]; ok {
		return nil, service.Errorf(service.CodeInvalidArgument, "map %q already exists", cfg.ID)
	}
	return f.SaveMap(cfg)
}

func (f *fakeService) DeleteMap(id string) error { return nil }

func (f *fakeService) GetSession(id string) (*service.Session, error) {
	if s, ok := f.sessions[id]; ok {
		return s.Clone(), nil
	}
	return nil, service.Errorf(service.CodeNotFound, "session %q not found", id)
}

func (f *fakeService) SetProgram(id string, p engine.Program) (*service.Session, error) {
	return nil, f.err
}

func testMapConfig(id string) engine.MapConfig {
	return engine.MapConfig{
		ID: id, Name: id,
		Layout:         []string{"...", ">.*", "..."},
		MainTapeLength: 2, SubTapeLengths: []int{2},
	}
}

func mustMap(t *testing.T, id string) *engine.Map {
	t.Helper()
	m, err := engine.NewMap(testMapConfig(id))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var ge *gqlerror.Error
	if !errors.As(err, &ge) {
		t.Fatalf("want *gqlerror.Error, got %T (%v)", err, err)
	}
	code, _ := ge.Extensions["code"].(string)
	return code
}

func ctxWithKey(key string) context.Context {
	r, _ := http.NewRequest(http.MethodPost, "/graphql", nil)
	if key != "" {
		r.Header.Set("X-Admin-Key", key)
	}
	return context.WithValue(context.Background(), HTTPRequestKey{}, r)
}

func TestAdminGate(t *testing.T) {
	input := model.MapInput{ID: "new_map", Name: "n", Layout: testMapConfig("x").Layout}
	cases := []struct {
		name      string
		adminKey  string
		allowOpen string
		ctx       context.Context
		wantCode  string
	}{
		{"no key configured, not opted out", "", "", ctxWithKey(""), service.CodeForbidden},
		{"no key configured, opted out", "", "true", ctxWithKey(""), ""},
		{"key configured, header missing", "secret", "", ctxWithKey(""), service.CodeForbidden},
		{"key configured, wrong header", "secret", "", ctxWithKey("nope"), service.CodeForbidden},
		{"key configured, correct header", "secret", "", ctxWithKey("secret"), ""},
		{"key configured, no request in ctx", "secret", "", context.Background(), service.CodeForbidden},
		{"opt-out ignored when key configured", "secret", "true", ctxWithKey("nope"), service.CodeForbidden},
		{"key is prefix of header", "secret", "", ctxWithKey("secret-and-more"), service.CodeForbidden},
		{"header is prefix of key", "secret", "", ctxWithKey("secre"), service.CodeForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ADMIN_API_KEY", tc.adminKey)
			t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", tc.allowOpen)
			fs := &fakeService{maps: map[string]*engine.Map{"existing": mustMap(t, "existing")}}
			r := &Resolver{Service: fs}
			m := &mutationResolver{r}

			check := func(op string, err error) {
				t.Helper()
				if tc.wantCode == "" {
					if err != nil {
						t.Fatalf("%s: unexpected error %v", op, err)
					}
					return
				}
				if err == nil {
					t.Fatalf("%s: expected %s error, got none", op, tc.wantCode)
				}
				if got := codeOf(t, err); got != tc.wantCode {
					t.Fatalf("%s: code = %q, want %q", op, got, tc.wantCode)
				}
			}
			_, err := m.CreateMap(tc.ctx, input)
			check("createMap", err)
			upd := input
			upd.ID = "existing"
			_, err = m.UpdateMap(tc.ctx, upd)
			check("updateMap", err)
			_, err = m.DeleteMap(tc.ctx, "existing")
			check("deleteMap", err)
			_, err = m.ValidateMap(tc.ctx, input)
			check("validateMap", err)
			if tc.wantCode != "" && len(fs.saved) != 0 {
				t.Fatalf("map saved despite gate: %v", fs.saved)
			}
		})
	}
}

func TestCreateUpdateMapSemantics(t *testing.T) {
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("ADMIN_API_KEY", "")
	fs := &fakeService{maps: map[string]*engine.Map{"existing": mustMap(t, "existing")}}
	m := &mutationResolver{&Resolver{Service: fs}}
	ctx := context.Background()
	layout := testMapConfig("x").Layout

	if _, err := m.CreateMap(ctx, model.MapInput{ID: "existing", Name: "n", Layout: layout}); codeOf(t, err) != service.CodeInvalidArgument {
		t.Fatalf("create existing: err = %v", err)
	}
	if _, err := m.UpdateMap(ctx, model.MapInput{ID: "missing", Name: "n", Layout: layout}); codeOf(t, err) != service.CodeNotFound {
		t.Fatalf("update missing: err = %v", err)
	}
	// Invalid layout from the store surfaces as INVALID_ARGUMENT.
	if _, err := m.CreateMap(ctx, model.MapInput{ID: "bad", Name: "n", Layout: []string{"."}}); codeOf(t, err) != service.CodeInvalidArgument {
		t.Fatalf("create invalid: err = %v", err)
	}
	got, err := m.CreateMap(ctx, model.MapInput{ID: "fresh", Name: "n", Layout: layout})
	if err != nil || got.ID != "fresh" || got.Start.X != 0 || got.Start.Y != 1 || got.StartFacing != model.FacingRight {
		t.Fatalf("create fresh = %+v, %v", got, err)
	}
}

// An explicit empty subTapeLengths means zero subs; omitted means the engine
// default. The distinction must survive the input conversion.
func TestCreateMapSubTapeLengthsEmptyVsOmitted(t *testing.T) {
	t.Setenv("ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("ADMIN_API_KEY", "")
	layout := testMapConfig("x").Layout
	cases := []struct {
		name string
		in   []int
		want []int
	}{
		{"omitted", nil, []int{engine.DefaultSubTape}},
		{"empty", []int{}, []int{}},
		{"explicit", []int{4, 5}, []int{4, 5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &mutationResolver{&Resolver{Service: &fakeService{}}}
			got, err := m.CreateMap(context.Background(), model.MapInput{ID: "fresh", Name: "n", Layout: layout, SubTapeLengths: tc.in})
			if err != nil {
				t.Fatal(err)
			}
			if len(got.SubTapeLengths) != len(tc.want) {
				t.Fatalf("subTapeLengths = %v, want %v", got.SubTapeLengths, tc.want)
			}
			for i := range tc.want {
				if got.SubTapeLengths[i] != tc.want[i] {
					t.Fatalf("subTapeLengths = %v, want %v", got.SubTapeLengths, tc.want)
				}
			}
		})
	}
}

func TestToGQLErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"coded", service.Errorf(service.CodeSessionPlaying, "playing"), service.CodeSessionPlaying},
		{"wrapped coded", errors.Join(errors.New("ctx"), service.Errorf(service.CodeNotFound, "x")), service.CodeNotFound},
		{"map validation", &engine.ValidationError{Issues: []string{"bad"}}, service.CodeInvalidArgument},
		{"program error", &engine.ProgramError{}, service.CodeInvalidProgram},
		{"unknown", errors.New("disk on fire"), CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codeOf(t, toGQLError(tc.err)); got != tc.code {
				t.Fatalf("code = %q, want %q", got, tc.code)
			}
		})
	}
	if err := toGQLError(errors.New("secret detail")); err.Error() == "secret detail" {
		t.Fatal("internal error message leaked to client")
	}
	if toGQLError(nil) != nil {
		t.Fatal("nil must map to nil")
	}
}

func TestSetProgramPropagatesCode(t *testing.T) {
	fs := &fakeService{err: service.Errorf(service.CodeSessionPlaying, "session is playing")}
	m := &mutationResolver{&Resolver{Service: fs}}
	_, err := m.SetProgram(context.Background(), "s", model.ProgramInput{})
	if codeOf(t, err) != service.CodeSessionPlaying {
		t.Fatalf("err = %v", err)
	}
}

func TestQueryNotFoundIsNull(t *testing.T) {
	q := &queryResolver{&Resolver{Service: &fakeService{}}}
	if s, err := q.Session(context.Background(), "nope"); s != nil || err != nil {
		t.Fatalf("session = %v, %v; want nil,nil", s, err)
	}
	if m, err := q.Map(context.Background(), "nope"); m != nil || err != nil {
		t.Fatalf("map = %v, %v; want nil,nil", m, err)
	}
}

func TestSessionConversionDerivesTreatsRemaining(t *testing.T) {
	m := mustMap(t, "m")
	vm := engine.NewVMState(m)
	s := &service.Session{ID: "s", MapID: "m", Map: m, VM: vm, Seq: 3, BestSteps: 0}
	got := toSession(s)
	if len(got.VM.TreatsRemaining) != 1 || got.VM.TreatsRemaining[0].X != 2 || got.VM.TreatsRemaining[0].Y != 1 {
		t.Fatalf("treatsRemaining = %+v", got.VM.TreatsRemaining)
	}
	if got.BestSteps != nil {
		t.Fatal("bestSteps must be null before first win")
	}
	if got.VM.LossReason != nil || got.LastEvent != nil {
		t.Fatal("lossReason/lastEvent must be null")
	}
	if got.Seq != 3 || got.VM.Facing != model.FacingRight {
		t.Fatalf("seq/facing = %d/%s", got.Seq, got.VM.Facing)
	}
}

func liveSession(t *testing.T, seq uint64) *service.Session {
	t.Helper()
	m := mustMap(t, "m")
	return &service.Session{ID: "s1", MapID: "m", Map: m, VM: engine.NewVMState(m), Seq: seq}
}

func TestForwardInitialThenUpdatesAndSeqFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan *service.Session, 4)
	out := forwardSessionUpdates(ctx, liveSession(t, 5), updates, 4)

	recv := func() *model.SessionUpdate {
		t.Helper()
		select {
		case u, ok := <-out:
			if !ok {
				t.Fatal("channel closed early")
			}
			return u
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
			return nil
		}
	}
	if u := recv(); u.Seq != 5 {
		t.Fatalf("initial seq = %d, want 5", u.Seq)
	}
	updates <- liveSession(t, 5) // duplicate of initial: dropped
	updates <- liveSession(t, 4) // stale: dropped
	updates <- liveSession(t, 6)
	if u := recv(); u.Seq != 6 || u.Session.Seq != 6 {
		t.Fatalf("next seq = %d, want 6", u.Seq)
	}
}

func goroutinesSettle(t *testing.T, base int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= base {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutine leak: %d running, baseline %d", runtime.NumGoroutine(), base)
}

func TestForwardEndsOnCancelWithoutReader(t *testing.T) {
	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan *service.Session)
	// Buffer 1 and nobody reading: the initial send fits, the next blocks.
	out := forwardSessionUpdates(ctx, liveSession(t, 1), updates, 1)
	go func() { updates <- liveSession(t, 2) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	for range out { // drains until close; close proves the goroutine exited
	}
	goroutinesSettle(t, base)
}

func TestSubscriptionResolverEmitsSnapshotAndStopsOnCancel(t *testing.T) {
	hub := websocket.NewHub()
	fs := &fakeService{sessions: map[string]*service.Session{"s1": liveSession(t, 2)}}
	sub := &subscriptionResolver{&Resolver{Service: fs, Hub: hub}}

	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := sub.SessionUpdated(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case u := <-ch:
		if u.Seq != 2 || u.Session.ID != "s1" {
			t.Fatalf("initial = %+v", u)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no initial snapshot")
	}
	next := liveSession(t, 3)
	hub.Broadcast(next)
	select {
	case u := <-ch:
		if u.Seq != 3 {
			t.Fatalf("update seq = %d", u.Seq)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no hub update forwarded")
	}
	cancel()
	for range ch {
	}
	goroutinesSettle(t, base)

	if _, err := sub.SessionUpdated(context.Background(), "missing"); codeOf(t, err) != service.CodeNotFound {
		t.Fatalf("missing session err = %v", err)
	}
}
