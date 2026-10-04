package engine

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

func mustMap(t *testing.T, cfg MapConfig) *Map {
	t.Helper()
	m, err := NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func prog(m *Map, main string, subs ...string) Program {
	conv := func(s string, n int) []Instruction {
		out := make([]Instruction, n)
		for i := range out {
			out[i] = Empty
		}
		for i, c := range s {
			switch c {
			case 'F':
				out[i] = MoveForward
			case 'L':
				out[i] = TurnLeft
			case 'R':
				out[i] = TurnRight
			case '1':
				out[i] = CallSub1
			case '2':
				out[i] = CallSub2
			case '3':
				out[i] = CallSub3
			case '_':
			}
		}
		return out
	}
	p := Program{Main: conv(main, m.MainTapeLength), Subs: make([][]Instruction, len(m.SubTapeLengths))}
	for i, n := range m.SubTapeLengths {
		var s string
		if i < len(subs) {
			s = subs[i]
		}
		p.Subs[i] = conv(s, n)
	}
	return p
}

var (
	straight = MapConfig{ID: "straight_line", Name: "s", MainTapeLength: 8, SubTapeLengths: []int{4}, Layout: []string{
		"..........", "..........", "..........", "..#.......", ".......#..", "..........", "..........", "..........", ".>......*.", ".........."}}
	symmetric = MapConfig{ID: "symmetric_paths", Name: "s", MainTapeLength: 13, SubTapeLengths: []int{6, 6, 6}, Layout: []string{
		"..........", ".*.......*", ".....#....", "..........", "..........", "..#..^..#.", "..........", "..........", ".....#....", ".*.......*"}}
	zigzag = MapConfig{ID: "zigzag_path", Name: "z", MainTapeLength: 6, SubTapeLengths: []int{4, 4, 4}, Layout: []string{
		"...*..*...", "..........", "..........", ">..*..*.*.", "..........", "..........", "..........", "..........", "..........", ".........."}}
	corridor = MapConfig{ID: "long_corridor", Name: "c", MainTapeLength: 3, SubTapeLengths: []int{2}, MaxSteps: 60, MaxCallDepth: 20, AllowRecursion: true, Layout: []string{
		"................", ">..............*", "................"}}
	staircase = MapConfig{ID: "staircase", Name: "c", MainTapeLength: 3, SubTapeLengths: []int{5}, MaxSteps: 60, MaxCallDepth: 10, AllowRecursion: true, Layout: []string{
		".........*", "..........", "..........", "..........", "..........", "....*.....", "..........", "..........", "..........", ">........."}}
)

func TestGoldenSolutions(t *testing.T) {
	cases := []struct {
		name  string
		cfg   MapConfig
		p     func(*Map) Program
		steps int
	}{
		{"straight", straight, func(m *Map) Program { return prog(m, "FFFFFFF") }, 7},
		{"symmetric", symmetric, func(m *Map) Program {
			return prog(m, "3R2L3L2212L22", "LFFFF", "FFFF", "FF")
		}, 46},
		{"zigzag", zigzag, func(m *Map) Program {
			return prog(m, "1L1R23", "FFF", "FFFR", "1LFF")
		}, 23},
		{"corridor", corridor, func(m *Map) Program { return prog(m, "1", "F1") }, 30},
		{"staircase", staircase, func(m *Map) Program { return prog(m, "1", "FLFR1") }, 44},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := mustMap(t, c.cfg)
			p := c.p(m)
			if err := ValidateProgram(m, p); err != nil {
				t.Fatal(err)
			}
			st := Run(m, p)
			if st.Status != StatusWon {
				t.Fatalf("status %s/%s steps %d pos %v", st.Status, st.LossReason, st.Steps, st.Pos)
			}
			if c.steps != 0 && st.Steps != c.steps {
				t.Fatalf("steps = %d, want %d", st.Steps, c.steps)
			}
		})
	}
}

func TestStrictRecursion(t *testing.T) {
	cfg := corridor
	cfg.AllowRecursion = false
	m := mustMap(t, cfg)
	st := Run(m, prog(m, "1", "F1"))
	if st.Status != StatusLost || st.LossReason != LossProgramEnded || st.Steps != 3 {
		t.Fatalf("got %s/%s steps %d", st.Status, st.LossReason, st.Steps)
	}
	// self call from main->sub1->sub1 is a no-op that still counts a step.
	evs, _ := Simulate(m, prog(m, "1", "1F"), NewVMState(m), 0)
	if len(evs) != 3 || evs[1].Instruction != CallSub1 || len(evs[1].CallStack) != 2 || evs[2].Instruction != MoveForward {
		t.Fatalf("events %+v", evs)
	}
}

func TestMutualRecursionBlocked(t *testing.T) {
	cfg := zigzag
	m := mustMap(t, cfg)
	// sub1 -> sub2 -> sub1 : inner call to sub1 is a no-op.
	evs, st := Simulate(m, prog(m, "1", "2F", "1F"), NewVMState(m), 0)
	if st.Status != StatusLost || st.LossReason != LossProgramEnded {
		t.Fatalf("got %s/%s", st.Status, st.LossReason)
	}
	// main CALL1, CALL2, CALL1(noop), F, F => 5 steps
	if len(evs) != 5 {
		t.Fatalf("events=%d", len(evs))
	}
}

func TestCallDepth(t *testing.T) {
	c := corridor
	c.MaxCallDepth = 10
	m := mustMap(t, c)
	st := Run(m, prog(m, "1", "F1"))
	if st.Status != StatusLost || st.LossReason != LossCallDepth || st.Steps != 21 {
		t.Fatalf("got %s/%s steps %d", st.Status, st.LossReason, st.Steps)
	}
	// depth exactly == max is fine: main->sub1 is depth 1
	c.MaxCallDepth = 1
	m = mustMap(t, c)
	st = Run(m, prog(m, "1", "FF"))
	if st.LossReason == LossCallDepth {
		t.Fatalf("depth 1 should be allowed")
	}
}

func TestStepLimitAndWinTie(t *testing.T) {
	c := straight
	c.MaxSteps = 5
	m := mustMap(t, c)
	st := Run(m, prog(m, "FFFFFFF"))
	if st.LossReason != LossStepLimit || st.Steps != 5 {
		t.Fatalf("got %s/%s %d", st.Status, st.LossReason, st.Steps)
	}
	c.MaxSteps = 7
	m = mustMap(t, c)
	if st = Run(m, prog(m, "FFFFFFF")); st.Status != StatusWon {
		t.Fatalf("win must beat step cap on same step, got %s/%s", st.Status, st.LossReason)
	}
}

func TestEmptySkipAndProgramEnded(t *testing.T) {
	m := mustMap(t, straight)
	p := prog(m, "L__R__F")
	evs, st := Simulate(m, p, NewVMState(m), 0)
	if len(evs) != 3 || st.Steps != 3 {
		t.Fatalf("EMPTY must not count: %d events", len(evs))
	}
	if st.Status != StatusLost || st.LossReason != LossProgramEnded {
		t.Fatalf("got %s/%s", st.Status, st.LossReason)
	}
	// eager: terminal on the same step as the last real instruction
	if last := evs[len(evs)-1]; last.Status != StatusLost {
		t.Fatalf("last event status %s", last.Status)
	}
	// all-empty program ends immediately
	vm := NewVMState(m)
	empty := EmptyProgram(m)
	vm.Settle(&empty)
	if vm.Status != StatusLost || vm.LossReason != LossProgramEnded {
		t.Fatalf("got %s", vm.Status)
	}
	if _, err := vm.Step(m, &empty); err != ErrTerminal {
		t.Fatalf("want ErrTerminal, got %v", err)
	}
}

func TestBlockedMoves(t *testing.T) {
	big := straight
	big.MainTapeLength = 12
	m := mustMap(t, big)
	// facing up from (1,8) into bounds then rock ahead at (2,3): turn right at row... walk to the rock.
	evs, st := Simulate(m, prog(m, "LFFFFFFFFF"), NewVMState(m), 0) // up from (1,8): reaches (1,0), 8th F blocked by bounds
	if st.Pos != (Position{1, 0}) {
		t.Fatalf("pos %v", st.Pos)
	}
	if !evs[len(evs)-1].Blocked || evs[len(evs)-1].From != evs[len(evs)-1].To {
		t.Fatalf("expected blocked move: %+v", evs[len(evs)-1])
	}
	// rock: start (1,8) go to row 3 then right into (2,3).
	m2 := mustMap(t, big)
	_, st = Simulate(m2, prog(m2, "LFFFFFRF"), NewVMState(m2), 0)
	if st.Pos != (Position{1, 3}) {
		t.Fatalf("rock should block, pos %v", st.Pos)
	}
}

func TestTapeEndReturnToCaller(t *testing.T) {
	m := mustMap(t, symmetric)
	evs, _ := Simulate(m, prog(m, "1F", "FF"), NewVMState(m), 0)
	var got []int
	for _, e := range evs {
		got = append(got, e.TapeIndex)
	}
	want := []int{-1, 0, 0, -1}
	if len(got) != 4 {
		t.Fatalf("tapes %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tapes %v want %v", got, want)
		}
	}
}

func TestValidateProgram(t *testing.T) {
	m := mustMap(t, straight)
	bad := []Program{
		{Main: make([]Instruction, 3), Subs: [][]Instruction{make([]Instruction, 4)}},
		prog(m, ""),
	}
	bad[1].Subs = nil
	p3 := prog(m, "")
	p3.Main[0] = CallSub2 // map has 1 sub
	p4 := prog(m, "")
	p4.Main[0] = "BOGUS"
	bad = append(bad, p3, p4)
	for i, p := range bad {
		if ValidateProgram(m, p) == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
	if err := ValidateProgram(m, prog(m, "FFF")); err != nil {
		t.Fatal(err)
	}
}

func TestNewMapValidation(t *testing.T) {
	cases := map[string]MapConfig{
		"bad id":      {ID: "../x", Name: "n", Layout: straight.Layout},
		"no start":    {ID: "a", Name: "n", Layout: []string{"...", "...", ".*."}},
		"two starts":  {ID: "a", Name: "n", Layout: []string{">..", "..<", ".*."}},
		"no treat":    {ID: "a", Name: "n", Layout: []string{">..", "...", "..."}},
		"ragged":      {ID: "a", Name: "n", Layout: []string{">..", "..", ".*."}},
		"bad glyph":   {ID: "a", Name: "n", Layout: []string{">x.", "...", ".*."}},
		"too small":   {ID: "a", Name: "n", Layout: []string{">*", ".."}},
		"steps cap":   {ID: "a", Name: "n", MaxSteps: MaxMaxSteps + 1, Layout: straight.Layout},
		"four subs":   {ID: "a", Name: "n", SubTapeLengths: []int{1, 1, 1, 1}, Layout: straight.Layout},
		"zero length": {ID: "a", Name: "n", SubTapeLengths: []int{0}, Layout: straight.Layout},
	}
	for name, c := range cases {
		if _, err := NewMap(c); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	m, err := NewMap(MapConfig{ID: "ok", Name: "n", Layout: []string{">.*", "...", "..."}})
	if err != nil {
		t.Fatal(err)
	}
	if m.MainTapeLength != 10 || len(m.SubTapeLengths) != 1 || m.SubTapeLengths[0] != 10 || m.MaxSteps != 200 || m.MaxCallDepth != 16 || m.AllowRecursion {
		t.Fatalf("defaults wrong: %+v", m)
	}
}

func BenchmarkSimulateCorridor(b *testing.B) {
	m, _ := NewMap(corridor)
	p := prog(m, "1", "F1")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		st := NewVMState(m)
		for !st.Status.Terminal() {
			_ = st.step(m, &p, nil)
		}
	}
}

func TestNewMapHugeRaggedRowRejectedWithoutAllocating(t *testing.T) {
	huge := strings.Repeat(".", 50_000_000)
	cfg := MapConfig{ID: "a", Name: "n", Layout: []string{huge, "...", "..."}}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := NewMap(cfg)
	runtime.ReadMemStats(&after)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	// W*H grids would be >=150 MB (bool + int slices); stay far below.
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 10<<20 {
		t.Fatalf("NewMap allocated %d bytes for an out-of-bounds layout", grew)
	}
}

func TestSubIndexCanonicalOnly(t *testing.T) {
	for in, want := range map[Instruction]int{"CALL_SUB_1": 0, "CALL_SUB_2": 1, "CALL_SUB_3": 2} {
		if k, ok := in.SubIndex(); !ok || k != want {
			t.Errorf("%s: got (%d,%v), want (%d,true)", in, k, ok, want)
		}
	}
	for _, in := range []Instruction{"CALL_SUB_01", "CALL_SUB_001", "CALL_SUB_0", "CALL_SUB_4", "CALL_SUB_10", "CALL_SUB_", "CALL_SUB_-1", "CALL_SUB_ 1", "CALL_SUB_18446744073709551617"} {
		if _, ok := in.SubIndex(); ok {
			t.Errorf("%q must not be a sub call", in)
		}
		if in.Valid() {
			t.Errorf("%q must not be valid", in)
		}
	}
	m := mustMap(t, straight)
	p := EmptyProgram(m)
	p.Main[0] = "CALL_SUB_01"
	if err := ValidateProgram(m, p); err == nil {
		t.Fatal("ValidateProgram must reject CALL_SUB_01")
	}
}
