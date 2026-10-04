// Package validate checks maps (structure, treat reachability) and their
// reference solutions.
package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
)

// Result is the structured outcome of Check.
type Result struct {
	Valid             bool              `json:"valid"`
	Issues            []string          `json:"issues"`
	UnreachableTreats []engine.Position `json:"unreachableTreats"`
}

// Check validates cfg structurally (engine.NewMap) and verifies every treat
// is reachable from the start ignoring facing (a necessary condition).
func Check(cfg engine.MapConfig) Result {
	res := Result{Issues: []string{}, UnreachableTreats: []engine.Position{}}
	m, err := engine.NewMap(cfg)
	if err != nil {
		var ve *engine.ValidationError
		if errors.As(err, &ve) {
			res.Issues = append(res.Issues, ve.Issues...)
		} else {
			res.Issues = append(res.Issues, err.Error())
		}
		return res
	}
	res.UnreachableTreats = UnreachableTreats(m)
	for _, p := range res.UnreachableTreats {
		res.Issues = append(res.Issues, fmt.Sprintf("treat at (%d,%d) is unreachable from the start", p.X, p.Y))
	}
	res.Valid = len(res.Issues) == 0
	return res
}

// UnreachableTreats returns the treats not reachable from the start by
// 4-neighbour moves through non-rock cells.
func UnreachableTreats(m *engine.Map) []engine.Position {
	seen := make([]bool, m.Width*m.Height)
	queue := []engine.Position{m.Start}
	seen[m.Start.Y*m.Width+m.Start.X] = true
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			x, y := p.X+d[0], p.Y+d[1]
			if m.IsRock(x, y) || seen[y*m.Width+x] {
				continue
			}
			seen[y*m.Width+x] = true
			queue = append(queue, engine.Position{X: x, Y: y})
		}
	}
	out := []engine.Position{}
	for _, t := range m.Treats {
		if !seen[t.Y*m.Width+t.X] {
			out = append(out, t)
		}
	}
	return out
}

// CheckSolution verifies p is a valid program for m and simulates to WON
// within the map's limits.
func CheckSolution(m *engine.Map, p engine.Program) error {
	if err := engine.ValidateProgram(m, p); err != nil {
		return fmt.Errorf("invalid program: %w", err)
	}
	st := engine.Run(m, p)
	if st.Status != engine.StatusWon {
		return fmt.Errorf("solution does not win: status=%s loss=%s steps=%d treats left=%d",
			st.Status, st.LossReason, st.Steps, st.TreatsLeft)
	}
	return nil
}

// CheckRecursionRequired verifies that for a recursion-enabled map the
// solution does NOT win when recursion is forbidden.
func CheckRecursionRequired(m *engine.Map, p engine.Program) error {
	if !m.AllowRecursion {
		return nil
	}
	cfg := m.Config()
	cfg.AllowRecursion = false
	nr, err := engine.NewMap(cfg)
	if err != nil {
		return err
	}
	if st := engine.Run(nr, p); st.Status == engine.StatusWon {
		return errors.New("map allows recursion but its solution also wins with allow_recursion=false")
	}
	return nil
}

// LoadSolution reads <dir>/<mapID>.json as a Program.
func LoadSolution(dir, mapID string) (engine.Program, error) {
	var p engine.Program
	if !engine.ValidID(mapID) {
		return p, fmt.Errorf("invalid map id %q", mapID)
	}
	path := filepath.Join(dir, mapID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("solution for map %q: %w", mapID, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return engine.Program{}, fmt.Errorf("%s: %w", path, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return engine.Program{}, fmt.Errorf("%s: unexpected data after JSON value", path)
	}
	return p, nil
}
