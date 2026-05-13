package core

import (
	"path/filepath"
	"testing"
)

func TestSlugifyName(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"already-a-slug", "sodium", "sodium"},
		{"lowercases", "Sodium", "sodium"},
		{"replaces spaces with dashes", "Iris Shaders", "iris-shaders"},
		{"strips parenthesized suffix", "A Man With Plushies (AMWPlushies)", "a-man-with-plushies"},
		{"strips ' - subtitle' suffix", "Mod Name - Forge Edition", "mod-name"},
		{"collapses runs of dashes", "Foo - - Bar", "foo"},
		{"removes leading and trailing dashes", "  spaced name  ", "spaced-name"},
		{"keeps digits", "BetterF3 v4", "betterf3-v4"},
		{"drops non-alphanumeric characters", "Re:Source Pack!", "re-source-pack"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SlugifyName(tc.input)

			if got != tc.want {
				t.Errorf("SlugifyName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestModPathAccessors(t *testing.T) {
	var m Mod

	got := m.SetMetaPath(filepath.Join("mods", "sodium.pw.toml"))
	want := filepath.Join("mods", "sodium.pw.toml")

	if got != want {
		t.Errorf("SetMetaPath returned %q, want %q", got, want)
	}

	if got := m.GetFilePath(); got != want {
		t.Errorf("GetFilePath = %q, want %q", got, want)
	}

	m.FileName = "sodium-fabric-mc1.20.1-0.5.3.jar"
	gotDest := m.GetDestFilePath()
	wantDest := filepath.Join("mods", "sodium-fabric-mc1.20.1-0.5.3.jar")

	if gotDest != wantDest {
		t.Errorf("GetDestFilePath = %q, want %q", gotDest, wantDest)
	}
}

func TestGetParsedUpdateData(t *testing.T) {
	m := Mod{
		updateData: map[string]interface{}{
			"curseforge": map[string]int{"file-id": 12345},
		},
	}

	got, ok := m.GetParsedUpdateData("curseforge")
	if !ok {
		t.Fatalf("expected GetParsedUpdateData(curseforge) to find entry")
	}

	if m, isMap := got.(map[string]int); !isMap || m["file-id"] != 12345 {
		t.Errorf("got %#v, want map with file-id=12345", got)
	}

	if _, ok := m.GetParsedUpdateData("modrinth"); ok {
		t.Errorf("expected miss for modrinth, got hit")
	}
}
