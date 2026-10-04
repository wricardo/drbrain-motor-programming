package engine

import (
	"encoding/json"
	"reflect"
	"testing"
)

// A map with no subs ("sub_tape_lengths": []) must survive Clone and a JSON
// round trip; omitting the field still selects the default single sub.
func TestZeroSubMapRoundTrip(t *testing.T) {
	cfg := MapConfig{ID: "nosubs", Name: "n", Layout: []string{".....", ">...*", "....."}, SubTapeLengths: []int{}}
	m, err := NewMap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.SubTapeLengths) != 0 {
		t.Fatalf("NewMap: subs = %v, want none", m.SubTapeLengths)
	}
	if c := m.Clone(); len(c.SubTapeLengths) != 0 {
		t.Fatalf("Clone: subs = %v, want none", c.SubTapeLengths)
	}

	data, err := json.Marshal(m.Config())
	if err != nil {
		t.Fatal(err)
	}
	var back MapConfig
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	m2, err := NewMap(back)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Config(), m2.Config()) || len(m2.SubTapeLengths) != 0 {
		t.Fatalf("JSON round trip changed the map: %s -> %+v", data, m2.Config())
	}

	cfg.SubTapeLengths = nil
	if d, err := NewMap(cfg); err != nil || len(d.SubTapeLengths) != 1 || d.SubTapeLengths[0] != DefaultSubTape {
		t.Fatalf("omitted sub_tape_lengths must default to [%d]: %v %v", DefaultSubTape, d, err)
	}
}
