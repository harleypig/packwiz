package core

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestGetMCVersion(t *testing.T) {
	t.Run("returns version when set", func(t *testing.T) {
		p := Pack{Versions: map[string]string{"minecraft": "1.20.1"}}

		got, err := p.GetMCVersion()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "1.20.1" {
			t.Errorf("got %q, want %q", got, "1.20.1")
		}
	})

	t.Run("returns error when minecraft version absent", func(t *testing.T) {
		p := Pack{Versions: map[string]string{"fabric": "0.15.0"}}

		if _, err := p.GetMCVersion(); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetSupportedMCVersions(t *testing.T) {
	// Use a clean viper state for the duration of this test.
	t.Cleanup(func() { viper.Set("acceptable-game-versions", nil) })
	viper.Set("acceptable-game-versions", nil)

	t.Run("returns just the configured version when no extras", func(t *testing.T) {
		p := Pack{Versions: map[string]string{"minecraft": "1.20.1"}}

		got, err := p.GetSupportedMCVersions()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(got, []string{"1.20.1"}) {
			t.Errorf("got %v, want [1.20.1]", got)
		}
	})

	t.Run("appends acceptable-game-versions and dedups so main version is last", func(t *testing.T) {
		viper.Set("acceptable-game-versions", []string{"1.19.4", "1.20.1", "1.20"})
		p := Pack{Versions: map[string]string{"minecraft": "1.20.1"}}

		got, err := p.GetSupportedMCVersions()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 1.20.1 appears in extras AND as main; the dedup logic keeps the
		// later occurrence, so the main version ends up last.
		want := []string{"1.19.4", "1.20", "1.20.1"}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("returns error when minecraft version absent", func(t *testing.T) {
		viper.Set("acceptable-game-versions", nil)
		p := Pack{Versions: map[string]string{}}

		if _, err := p.GetSupportedMCVersions(); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetPackName(t *testing.T) {
	cases := []struct {
		name    string
		pack    Pack
		want    string
	}{
		{"empty name falls back to export", Pack{}, "export"},
		{"name only", Pack{Name: "MyPack"}, "MyPack"},
		{"name plus version", Pack{Name: "MyPack", Version: "1.0"}, "MyPack-1.0"},
		{"version without name still falls back", Pack{Version: "1.0"}, "export"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pack.GetPackName(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetCompatibleLoaders(t *testing.T) {
	cases := []struct {
		name     string
		versions map[string]string
		want     []string
	}{
		{"quilt implies fabric compat", map[string]string{"quilt": "1"}, []string{"quilt", "fabric"}},
		{"fabric alone", map[string]string{"fabric": "1"}, []string{"fabric"}},
		{"neoforge implies forge compat", map[string]string{"neoforge": "1"}, []string{"neoforge", "forge"}},
		{"forge alone", map[string]string{"forge": "1"}, []string{"forge"}},
		{"quilt + neoforge", map[string]string{"quilt": "1", "neoforge": "1"}, []string{"quilt", "fabric", "neoforge", "forge"}},
		{"none", map[string]string{}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Pack{Versions: tc.versions}

			got := p.GetCompatibleLoaders()

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetLoaders(t *testing.T) {
	cases := []struct {
		name     string
		versions map[string]string
		want     []string
	}{
		{"single fabric", map[string]string{"fabric": "1"}, []string{"fabric"}},
		{"quilt does NOT imply fabric here", map[string]string{"quilt": "1"}, []string{"quilt"}},
		{"all four declared", map[string]string{"quilt": "1", "fabric": "1", "neoforge": "1", "forge": "1"}, []string{"quilt", "fabric", "neoforge", "forge"}},
		{"none", map[string]string{}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Pack{Versions: tc.versions}

			got := p.GetLoaders()

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
