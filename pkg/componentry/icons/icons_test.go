package icons_test

import (
	"sync"
	"testing"

	"github.com/go-sum/componentry/icons"
)

func TestRegistry_Register_Resolve(t *testing.T) {
	r := icons.NewRegistry()

	// Unregistered key returns zero Ref and false.
	ref, ok := r.Resolve(icons.ChevronDown)
	if ok {
		t.Errorf("Resolve unregistered key: got ok=true, want false")
	}
	if ref != (icons.Ref{}) {
		t.Errorf("Resolve unregistered key: got %+v, want zero Ref", ref)
	}

	// Register then resolve returns the correct Ref and true.
	want := icons.Ref{Sprite: "/icons.svg", ID: "chevron-down"}
	r.Register(icons.ChevronDown, want)

	got, ok := r.Resolve(icons.ChevronDown)
	if !ok {
		t.Errorf("Resolve registered key: got ok=false, want true")
	}
	if got != want {
		t.Errorf("Resolve registered key: got %+v, want %+v", got, want)
	}
}

func TestRegistry_Register_overwrite(t *testing.T) {
	r := icons.NewRegistry()
	first := icons.Ref{Sprite: "/a.svg", ID: "icon-a"}
	second := icons.Ref{Sprite: "/b.svg", ID: "icon-b"}

	r.Register(icons.Close, first)
	r.Register(icons.Close, second)

	got, ok := r.Resolve(icons.Close)
	if !ok {
		t.Fatalf("Resolve after overwrite: got ok=false")
	}
	if got != second {
		t.Errorf("Register overwrite: got %+v, want %+v", got, second)
	}
}

func TestRegistry_RegisterSet(t *testing.T) {
	r := icons.NewRegistry()
	set := map[icons.Key]icons.Ref{
		icons.ChevronLeft:  {Sprite: "/s.svg", ID: "chevron-left"},
		icons.ChevronRight: {Sprite: "/s.svg", ID: "chevron-right"},
		icons.ThemeLight:   {Sprite: "/s.svg", ID: "theme-light"},
	}
	r.RegisterSet(set)

	for key, want := range set {
		got, ok := r.Resolve(key)
		if !ok {
			t.Errorf("RegisterSet: key %q not found", key)
			continue
		}
		if got != want {
			t.Errorf("RegisterSet key %q: got %+v, want %+v", key, got, want)
		}
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	r := icons.NewRegistry()
	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent writers
	for i := range goroutines {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r.Register(icons.ChevronDown, icons.Ref{Sprite: "/s.svg", ID: "v"})
			_ = n
		}(i)
	}

	// Concurrent readers
	for i := range goroutines {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r.Resolve(icons.ChevronDown)
			_ = n
		}(i)
	}

	wg.Wait()
	// If no race was detected, the test passes.
}

