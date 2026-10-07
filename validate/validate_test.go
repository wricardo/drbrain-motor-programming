package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
)

const (
	mapsDir = "../maps"
	solDir  = "../solutions"
)

// requireSolutions skips tests that need the reference solutions, which are
// kept out of the public repo (solutions/ is gitignored).
func requireSolutions(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(solDir); err != nil {
		t.Skipf("reference solutions not present in %s", solDir)
	}
}

func loadMap(t *testing.T, id string) *engine.Map {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(mapsDir, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg engine.MapConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	m, err := engine.NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func withCfg(t *testing.T, m *engine.Map, f func(*engine.MapConfig)) *engine.Map {
	t.Helper()
	cfg := m.Config()
	f(&cfg)
	out, err := engine.NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEveryMapSolutionWinsWithGoldenSteps(t *testing.T) {
	requireSolutions(t)
	golden := map[string]int{
		"straight_line": 7, "turn_challenge": 17, "rock_obstacle": 19, "symmetric_paths": 46,
		"zigzag_path": 23, "long_corridor": 30, "staircase": 44, "serpentine": 80,
		"powers": 32, "pinwheel": 48, "bump_spiral": 83,
		"comb": 65, "diamond_ring": 77, "ladder": 57, "mountain_range": 40,
		"nested_doubling": 30, "switchback": 42,
	}
	files, _ := filepath.Glob(filepath.Join(mapsDir, "*.json"))
	if len(files) != len(golden) {
		t.Fatalf("found %d maps, want %d", len(files), len(golden))
	}
	for id, steps := range golden {
		t.Run(id, func(t *testing.T) {
			m := loadMap(t, id)
			if res := Check(m.Config()); !res.Valid {
				t.Fatalf("Check: %v", res.Issues)
			}
			p, err := LoadSolution(solDir, id)
			if err != nil {
				t.Fatal(err)
			}
			if err := CheckSolution(m, p); err != nil {
				t.Fatal(err)
			}
			if err := CheckRecursionRequired(m, p); err != nil {
				t.Fatal(err)
			}
			if st := engine.Run(m, p); st.Steps != steps {
				t.Fatalf("steps=%d want %d", st.Steps, steps)
			}
		})
	}
}

func TestRecursionMapsBehavior(t *testing.T) {
	requireSolutions(t)
	cases := []struct {
		id          string
		noRecSteps  int
		shallow     int
		shallowDesc string
	}{
		{"long_corridor", 3, 10, "max_call_depth 10"},
		{"staircase", 6, 8, "max_call_depth 8"},
		{"bump_spiral", 11, 8, "max_call_depth 8"},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			m := loadMap(t, c.id)
			p, err := LoadSolution(solDir, c.id)
			if err != nil {
				t.Fatal(err)
			}
			nr := withCfg(t, m, func(cfg *engine.MapConfig) { cfg.AllowRecursion = false })
			st := engine.Run(nr, p)
			if st.Status != engine.StatusLost || st.LossReason != engine.LossProgramEnded || st.Steps != c.noRecSteps {
				t.Fatalf("no recursion: %s/%s after %d steps, want LOST/PROGRAM_ENDED after %d", st.Status, st.LossReason, st.Steps, c.noRecSteps)
			}
			sh := withCfg(t, m, func(cfg *engine.MapConfig) { cfg.MaxCallDepth = c.shallow })
			st = engine.Run(sh, p)
			if st.Status != engine.StatusLost || st.LossReason != engine.LossCallDepth {
				t.Fatalf("%s: %s/%s, want LOST/CALL_DEPTH", c.shallowDesc, st.Status, st.LossReason)
			}
		})
	}
}

// maxForwardMoves bounds the MOVE_FORWARDs a non-recursive program can run on
// a single-sub map: each main slot executes at most one sub tape (a sub cannot
// re-enter itself), so moves <= main slots * sub tape length.
func maxForwardMoves(m *engine.Map) int {
	maxSub := 1
	for _, n := range m.SubTapeLengths {
		if n > maxSub {
			maxSub = n
		}
	}
	return m.MainTapeLength * maxSub
}

func manhattan(a, b engine.Position) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// requiredMoves is a lower bound on forward moves: the farthest treat from the
// start (Manhattan distance).
func requiredMoves(m *engine.Map) int {
	best := 0
	for _, tr := range m.Treats {
		if d := manhattan(m.Start, tr); d > best {
			best = d
		}
	}
	return best
}

func TestRecursionMapsCountingBound(t *testing.T) {
	for _, id := range []string{"long_corridor", "staircase"} {
		t.Run(id, func(t *testing.T) {
			m := loadMap(t, id)
			if got, need := maxForwardMoves(m), requiredMoves(m); got >= need {
				t.Fatalf("non-recursive max forward moves %d >= required %d: recursion not provably needed", got, need)
			}
		})
	}
	// Sanity: the bound figures documented in the plan.
	if m := loadMap(t, "long_corridor"); maxForwardMoves(m) != 6 || requiredMoves(m) != 15 {
		t.Errorf("long_corridor bound = %d/%d, want 6/15", maxForwardMoves(m), requiredMoves(m))
	}
	if m := loadMap(t, "staircase"); maxForwardMoves(m) != 15 || requiredMoves(m) != 18 {
		t.Errorf("staircase bound = %d/%d, want 15/18", maxForwardMoves(m), requiredMoves(m))
	}
}

// minActions is the fewest TURN/MOVE actions (calls excluded) that collect every
// treat, found by BFS over (position, facing, collected-treats bitmask).
func minActions(m *engine.Map) int {
	type state struct {
		pos    engine.Position
		facing engine.Facing
		mask   int
	}
	dirs := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	treatBit := map[engine.Position]int{}
	for i, tr := range m.Treats {
		treatBit[tr] = 1 << i
	}
	full := 1<<len(m.Treats) - 1
	start := state{m.Start, m.StartFacing, 0}
	dist := map[state]int{start: 0}
	queue := []state{start}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if s.mask == full {
			return dist[s]
		}
		next := []state{{s.pos, (s.facing + 3) % 4, s.mask}, {s.pos, (s.facing + 1) % 4, s.mask}}
		d := dirs[s.facing]
		if np := (engine.Position{X: s.pos.X + d[0], Y: s.pos.Y + d[1]}); !m.IsRock(np.X, np.Y) {
			next = append(next, state{np, s.facing, s.mask | treatBit[np]})
		}
		for _, n := range next {
			if _, seen := dist[n]; !seen {
				dist[n] = dist[s] + 1
				queue = append(queue, n)
			}
		}
	}
	return -1
}

// maxLeafActions bounds the TURN/MOVE actions a non-recursive program can
// execute when subs nest at most `levels` deep (main -> sub -> sub ...). A sub
// can never re-enter itself or an active sub, so each level picks a different
// sub: ceiling = tape length * best ceiling of the level below.
func maxLeafActions(m *engine.Map, levels int) int {
	var tape func(n int, used uint, d int) int
	tape = func(n int, used uint, d int) int {
		best := 1
		if d > 0 {
			for s, l := range m.SubTapeLengths {
				if used&(1<<uint(s)) == 0 {
					if v := tape(l, used|1<<uint(s), d-1); v > best {
						best = v
					}
				}
			}
		}
		return n * best
	}
	return tape(m.MainTapeLength, 0, levels)
}

// TestNestingMapsAreProvablyNeeded: for each map, the minimum number of
// actions exceeds what a shallower program can execute, while the shipped
// solution (which nests deeper / recurses) still wins.
func TestNestingMapsAreProvablyNeeded(t *testing.T) {
	requireSolutions(t)
	cases := []struct {
		id        string
		minNeeded int // BFS minimum of TURN/MOVE actions
		levels    int // deepest nesting that is still too shallow
		ceiling   int // maxLeafActions at that depth
	}{
		{"serpentine", 61, 1, 49},
		{"powers", 19, 2, 18},
		{"pinwheel", 35, 1, 20},
		{"bump_spiral", 56, 1, 30}, // single sub: no-recursion nesting is at most one level
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			m := loadMap(t, c.id)
			if got := minActions(m); got != c.minNeeded {
				t.Fatalf("minimum actions = %d, want %d", got, c.minNeeded)
			}
			if got := maxLeafActions(m, c.levels); got != c.ceiling {
				t.Fatalf("ceiling at %d levels = %d, want %d", c.levels, got, c.ceiling)
			}
			if c.ceiling >= c.minNeeded {
				t.Fatalf("ceiling %d >= required %d: not provably needed", c.ceiling, c.minNeeded)
			}
			p, err := LoadSolution(solDir, c.id)
			if err != nil {
				t.Fatal(err)
			}
			if st := engine.Run(m, p); st.Status != engine.StatusWon {
				t.Fatalf("solution: %s/%s", st.Status, st.LossReason)
			}
		})
	}
}

// Serpentine and powers must not need recursion; bump_spiral must.
func TestRecursionFlags(t *testing.T) {
	for id, want := range map[string]bool{"serpentine": false, "powers": false, "pinwheel": false, "bump_spiral": true} {
		if got := loadMap(t, id).AllowRecursion; got != want {
			t.Errorf("%s allow_recursion = %v, want %v", id, got, want)
		}
	}
}

func TestCheckStructuralIssues(t *testing.T) {
	res := Check(engine.MapConfig{ID: "x", Name: "x", Layout: []string{"...", "...", "..."}})
	if res.Valid || len(res.Issues) == 0 {
		t.Fatalf("want invalid with issues, got %+v", res)
	}
}

func TestCheckUnreachableTreat(t *testing.T) {
	res := Check(engine.MapConfig{
		ID: "walled", Name: "w",
		Layout: []string{
			">.#.*",
			"..#..",
			"..#..",
		},
		MainTapeLength: 4, SubTapeLengths: []int{2},
	})
	if res.Valid {
		t.Fatal("want invalid")
	}
	if len(res.UnreachableTreats) != 1 || res.UnreachableTreats[0] != (engine.Position{X: 4, Y: 0}) {
		t.Fatalf("unreachable = %v", res.UnreachableTreats)
	}
}

func TestCheckSolutionFailures(t *testing.T) {
	m := loadMap(t, "straight_line")
	if err := CheckSolution(m, engine.EmptyProgram(m)); err == nil {
		t.Fatal("empty program must not pass")
	}
	if err := CheckSolution(m, engine.Program{Main: []engine.Instruction{engine.MoveForward}}); err == nil || !strings.Contains(err.Error(), "invalid program") {
		t.Fatalf("wrong tape lengths must be rejected, got %v", err)
	}
}

func TestCheckRecursionRequiredDetectsUnneeded(t *testing.T) {
	requireSolutions(t)
	m := loadMap(t, "straight_line")
	rec := withCfg(t, m, func(c *engine.MapConfig) { c.AllowRecursion = true })
	p, err := LoadSolution(solDir, "straight_line")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckRecursionRequired(rec, p); err == nil {
		t.Fatal("solution that wins without recursion must be flagged")
	}
}

func TestLoadSolutionRejectsBadID(t *testing.T) {
	if _, err := LoadSolution(solDir, "../maps/straight_line"); err == nil {
		t.Fatal("want error for traversal id")
	}
	if _, err := LoadSolution(solDir, "missing_map"); err == nil {
		t.Fatal("want error for missing solution")
	}
}

func TestLoadSolutionRejectsTrailingData(t *testing.T) {
	dir := t.TempDir()
	good := `{"main":["MOVE_FORWARD"],"subs":[]}`
	write := func(id, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok", good+"\n  \n")
	write("trail", good+` {"main":[]}`)
	write("junk", good+`}`)
	if _, err := LoadSolution(dir, "ok"); err != nil {
		t.Fatalf("trailing whitespace must be allowed: %v", err)
	}
	for _, id := range []string{"trail", "junk"} {
		if _, err := LoadSolution(dir, id); err == nil {
			t.Errorf("%s: trailing data must be rejected", id)
		}
	}
}
