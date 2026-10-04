package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
)

func cfg(id string) engine.MapConfig {
	return engine.MapConfig{
		ID: id, Name: "T",
		Layout:         []string{"....", ".>.*", "...."},
		MainTapeLength: 4, SubTapeLengths: []int{2},
	}
}

func TestSaveReloadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := m.Save(cfg("b_map"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Save(cfg("a_map")); err != nil {
		t.Fatal(err)
	}
	m2, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m2.Get("b_map")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Config(), saved.Config()) {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", got.Config(), saved.Config())
	}
	list := m2.List()
	if len(list) != 2 || list[0].ID != "a_map" || list[1].ID != "b_map" {
		t.Fatalf("List not sorted by id: %v", list)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 2 {
		t.Fatalf("expected only 2 files (no temp leftovers), got %d", len(ents))
	}
}

func TestGetNotFoundAndDelete(t *testing.T) {
	dir := t.TempDir()
	m, _ := NewManager(dir)
	var se *service.Error
	if _, err := m.Get("nope"); !errors.As(err, &se) || se.Code != service.CodeNotFound {
		t.Fatalf("want NOT_FOUND, got %v", err)
	}
	if _, err := m.Save(cfg("x")); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.json")); !os.IsNotExist(err) {
		t.Fatalf("file should be gone: %v", err)
	}
	if err := m.Delete("x"); !errors.As(err, &se) || se.Code != service.CodeNotFound {
		t.Fatalf("want NOT_FOUND on second delete, got %v", err)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "maps")
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../evil", "a/b", "..", "", "UPPER", strings.Repeat("a", 65)} {
		if _, err := m.Save(cfg(id)); err == nil {
			t.Errorf("Save(%q) succeeded", id)
		}
		if err := m.Delete(id); err == nil {
			t.Errorf("Delete(%q) succeeded", id)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, "evil.json")); !os.IsNotExist(err) {
		t.Fatal("file escaped maps dir")
	}
}

func TestSaveInvalidMapRejected(t *testing.T) {
	m, _ := NewManager(t.TempDir())
	bad := cfg("bad")
	bad.Layout = []string{"...", "...", "..."} // no start, no treat
	var se *service.Error
	if _, err := m.Save(bad); !errors.As(err, &se) || se.Code != service.CodeInvalidArgument {
		t.Fatalf("want INVALID_ARGUMENT, got %v", err)
	}
	if len(m.List()) != 0 {
		t.Fatal("invalid map stored")
	}
}

func TestInvalidFileFailsFast(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`{"id":"broken","name":"x","layout":["...","...","..."]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewManager(dir)
	if err == nil || !strings.Contains(err.Error(), "broken.json") {
		t.Fatalf("want error naming broken.json, got %v", err)
	}
}

func TestIDFilenameMismatchFailsFast(t *testing.T) {
	dir := t.TempDir()
	data, _ := os.ReadFile("../../maps/straight_line.json")
	if err := os.WriteFile(filepath.Join(dir, "other.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewManager(dir); err == nil || !strings.Contains(err.Error(), "other.json") {
		t.Fatalf("want mismatch error, got %v", err)
	}
}

func TestReturnedMapsAreCopies(t *testing.T) {
	m, _ := NewManager(t.TempDir())
	if _, err := m.Save(cfg("c")); err != nil {
		t.Fatal(err)
	}
	a, _ := m.Get("c")
	a.Name = "mutated"
	a.SubTapeLengths[0] = 99
	b, _ := m.Get("c")
	if b.Name != "T" || b.SubTapeLengths[0] != 2 {
		t.Fatal("store aliased returned map")
	}
}

func TestRepoMapsLoad(t *testing.T) {
	m, err := NewManager("../../maps")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(m.List()); n != 11 {
		t.Fatalf("want 11 maps, got %d", n)
	}
}

func TestCreateFailsIfExistsAndIsAtomic(t *testing.T) {
	dir := t.TempDir()
	m, _ := NewManager(dir)

	const n = 16
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		c := cfg("race")
		c.Name = fmt.Sprintf("N%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Create(c)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	ok := 0
	for err := range errs {
		if err == nil {
			ok++
			continue
		}
		var se *service.Error
		if !errors.As(err, &se) || se.Code != service.CodeInvalidArgument {
			t.Fatalf("want INVALID_ARGUMENT conflict, got %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d creates succeeded, want exactly 1", ok)
	}
	// Winner's content is what is stored and on disk.
	got, _ := m.Get("race")
	m2, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	disk, _ := m2.Get("race")
	if got.Name != disk.Name {
		t.Fatalf("memory %q != disk %q", got.Name, disk.Name)
	}
	// Save still overwrites.
	if _, err := m.Save(cfg("race")); err != nil {
		t.Fatal(err)
	}
}
