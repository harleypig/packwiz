package cmd

import (
	"testing"

	"github.com/packwiz/packwiz/core"
)

func mkMod(name string, pinned bool) *core.Mod {
	m := &core.Mod{Name: name}
	m.Pin = pinned
	return m
}

func modNames(mods []*core.Mod) []string {
	names := make([]string, len(mods))
	for i, m := range mods {
		names[i] = m.Name
	}
	return names
}

func TestFilterByPin(t *testing.T) {
	pinned := mkMod("pinned-mod", true)
	unpinned := mkMod("unpinned-mod", false)
	also := mkMod("also-pinned", true)
	mixed := []*core.Mod{pinned, unpinned, also}

	t.Run("no flags returns all mods unchanged", func(t *testing.T) {
		got, err := filterByPin(mixed, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != len(mixed) {
			t.Errorf("got %d mods, want %d", len(got), len(mixed))
		}
	})

	t.Run("pinned flag returns only pinned mods", func(t *testing.T) {
		input := []*core.Mod{pinned, unpinned, also}

		got, err := filterByPin(input, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 2 {
			t.Fatalf("got %d mods, want 2: %v", len(got), modNames(got))
		}

		for _, m := range got {
			if !m.Pin {
				t.Errorf("mod %q has Pin=false, want Pin=true", m.Name)
			}
		}
	})

	t.Run("unpinned flag returns only unpinned mods", func(t *testing.T) {
		input := []*core.Mod{pinned, unpinned, also}

		got, err := filterByPin(input, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("got %d mods, want 1: %v", len(got), modNames(got))
		}

		if got[0].Pin {
			t.Errorf("mod %q has Pin=true, want Pin=false", got[0].Name)
		}
	})

	t.Run("both flags returns error", func(t *testing.T) {
		_, err := filterByPin(mixed, true, true)
		if err == nil {
			t.Error("expected error when both --pinned and --unpinned are set, got nil")
		}
	})

	t.Run("pinned flag on empty list returns empty", func(t *testing.T) {
		got, err := filterByPin([]*core.Mod{}, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("got %d mods, want 0", len(got))
		}
	})

	t.Run("unpinned flag on all-pinned list returns empty", func(t *testing.T) {
		allPinned := []*core.Mod{mkMod("a", true), mkMod("b", true)}

		got, err := filterByPin(allPinned, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("got %d mods, want 0", len(got))
		}
	})

	t.Run("pinned flag on all-unpinned list returns empty", func(t *testing.T) {
		allUnpinned := []*core.Mod{mkMod("x", false), mkMod("y", false)}

		got, err := filterByPin(allUnpinned, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("got %d mods, want 0", len(got))
		}
	})
}
