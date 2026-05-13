package curseforge

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/packwiz/packwiz/core"
	"github.com/spf13/viper"
)

func TestGetCurseforgeVersion(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain release passes through unchanged", "1.20.1", "1.20.1"},
		{"pre-release suffix triggers -Snapshot", "1.18.1-pre1", "1.18.1-Snapshot"},
		{"rc suffix triggers -Snapshot", "1.20-rc1", "1.20-Snapshot"},
		{"22w24a maps to 1.19-Snapshot", "22w24a", "1.19-Snapshot"},
		{"20w20a maps to 1.16-Snapshot per the snapshot table", "20w20a", "1.16-Snapshot"},
		{"non-matching gibberish returns input unchanged", "definitely-not-a-version", "definitely-not-a-version"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getCurseforgeVersion(tc.input); got != tc.want {
				t.Errorf("getCurseforgeVersion(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGetCurseforgeVersions(t *testing.T) {
	got := getCurseforgeVersions([]string{"1.20.1", "1.18.1-pre1", "1.19.4"})
	want := []string{"1.20.1", "1.18.1-Snapshot", "1.19.4"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := getCurseforgeVersions(nil); len(got) != 0 {
		t.Errorf("nil input should yield empty slice, got %v", got)
	}
}

func TestParseSlugOrUrl(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantGame     string
		wantCategory string
		wantSlug     string
		wantFileID   uint32
	}{
		{
			name:         "curseforge.com URL with category",
			input:        "https://www.curseforge.com/minecraft/mc-mods/sodium",
			wantGame:     "minecraft",
			wantCategory: "mc-mods",
			wantSlug:     "sodium",
		},
		{
			name:         "curseforge.com URL with category and fileID",
			input:        "https://www.curseforge.com/minecraft/mc-mods/sodium/files/4567890",
			wantGame:     "minecraft",
			wantCategory: "mc-mods",
			wantSlug:     "sodium",
			wantFileID:   4567890,
		},
		{
			name:     "legacy minecraft.curseforge.com URL (no category)",
			input:    "https://minecraft.curseforge.com/projects/sodium",
			wantGame: "minecraft",
			wantSlug: "sodium",
		},
		{
			name:     "plain slug",
			input:    "sodium",
			wantSlug: "sodium",
		},
		{
			// Trailing punctuation breaks all three regexes; the function
			// returns zero values with no error.
			name: "non-matching input returns zero values",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			game, category, slug, fileID, err := parseSlugOrUrl(tc.input)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if game != tc.wantGame {
				t.Errorf("game = %q, want %q", game, tc.wantGame)
			}

			if category != tc.wantCategory {
				t.Errorf("category = %q, want %q", category, tc.wantCategory)
			}

			if slug != tc.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tc.wantSlug)
			}

			if fileID != tc.wantFileID {
				t.Errorf("fileID = %d, want %d", fileID, tc.wantFileID)
			}
		})
	}
}

func TestGetPathForFile(t *testing.T) {
	t.Cleanup(func() {
		viper.Set("meta-folder", "")
		viper.Set("meta-folder-base", "")
	})

	t.Run("known game + class folder mapping", func(t *testing.T) {
		viper.Set("meta-folder", "")
		viper.Set("meta-folder-base", "")
		// gameID 432 = Minecraft, classID 6 = mods.
		got := getPathForFile(432, 6, 0, "sodium")
		want := filepath.Join("mods", "sodium"+core.MetaExtension)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("unknown class falls back to categoryID mapping", func(t *testing.T) {
		viper.Set("meta-folder", "")
		viper.Set("meta-folder-base", "")
		// classID 999 is unknown, categoryID 5 = plugins.
		got := getPathForFile(432, 999, 5, "luckperms")
		want := filepath.Join("plugins", "luckperms"+core.MetaExtension)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("unknown game falls back to flat layout", func(t *testing.T) {
		viper.Set("meta-folder", "")
		viper.Set("meta-folder-base", "")
		got := getPathForFile(0, 0, 0, "sodium")
		want := filepath.Join(".", "sodium"+core.MetaExtension)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("explicit meta-folder wins over folder mapping", func(t *testing.T) {
		viper.Set("meta-folder", "custom-folder")
		viper.Set("meta-folder-base", "")
		// classID 6 would normally map to "mods"; explicit setting overrides.
		got := getPathForFile(432, 6, 0, "sodium")
		want := filepath.Join("custom-folder", "sodium"+core.MetaExtension)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("meta-folder-base is prepended", func(t *testing.T) {
		viper.Set("meta-folder", "")
		viper.Set("meta-folder-base", "packroot")
		got := getPathForFile(432, 6, 0, "sodium")
		want := filepath.Join("packroot", "mods", "sodium"+core.MetaExtension)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetSearchLoaderType(t *testing.T) {
	cases := []struct {
		name string
		vers map[string]string
		want modloaderType
	}{
		{"only fabric → Fabric", map[string]string{"fabric": "1"}, modloaderTypeFabric},
		{"only forge → Forge", map[string]string{"forge": "1"}, modloaderTypeForge},
		{"fabric + quilt → Any (multiple loaders)", map[string]string{"fabric": "1", "quilt": "1"}, modloaderTypeAny},
		{"fabric + forge → Any (mixed)", map[string]string{"fabric": "1", "forge": "1"}, modloaderTypeAny},
		{"only quilt → Any (no special case)", map[string]string{"quilt": "1"}, modloaderTypeAny},
		{"only neoforge → Any (no special case)", map[string]string{"neoforge": "1"}, modloaderTypeAny},
		{"empty → Any", map[string]string{}, modloaderTypeAny},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := core.Pack{Versions: tc.vers}

			if got := getSearchLoaderType(p); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFilterLoaderTypeIndex(t *testing.T) {
	cases := []struct {
		name          string
		packLoaders   []string
		modLoaderType modloaderType
		wantType      modloaderType
		wantOK        bool
	}{
		{
			name:          "empty packLoaders allows any file",
			packLoaders:   nil,
			modLoaderType: modloaderTypeForge,
			wantType:      modloaderTypeAny,
			wantOK:        true,
		},
		{
			name:          "Any always passes through",
			packLoaders:   []string{"fabric"},
			modLoaderType: modloaderTypeAny,
			wantType:      modloaderTypeAny,
			wantOK:        true,
		},
		{
			name:          "supported loader passes through with its type",
			packLoaders:   []string{"fabric"},
			modLoaderType: modloaderTypeFabric,
			wantType:      modloaderTypeFabric,
			wantOK:        true,
		},
		{
			name:          "unsupported loader is filtered out",
			packLoaders:   []string{"forge"},
			modLoaderType: modloaderTypeFabric,
			wantType:      modloaderTypeAny,
			wantOK:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotOK := filterLoaderTypeIndex(tc.packLoaders, tc.modLoaderType)

			if gotType != tc.wantType || gotOK != tc.wantOK {
				t.Errorf("got (%d, %v), want (%d, %v)", gotType, gotOK, tc.wantType, tc.wantOK)
			}
		})
	}
}

func TestFilterFileInfoLoaderIndex(t *testing.T) {
	cases := []struct {
		name        string
		packLoaders []string
		fileVers    []string
		wantType    modloaderType
		wantOK      bool
	}{
		{
			name:        "empty packLoaders allows any file",
			packLoaders: nil,
			fileVers:    []string{"Fabric", "1.20.1"},
			wantType:    modloaderTypeAny,
			wantOK:      true,
		},
		{
			name:        "file with matching loader returns that loader",
			packLoaders: []string{"fabric"},
			fileVers:    []string{"Fabric", "1.20.1"},
			wantType:    modloaderTypeFabric,
			wantOK:      true,
		},
		{
			name:        "file with non-matching loader is rejected",
			packLoaders: []string{"fabric"},
			fileVers:    []string{"Forge", "1.20.1"},
			wantType:    modloaderTypeAny,
			wantOK:      false,
		},
		{
			// Higher modloaderType index wins: Fabric (4) > Forge (1).
			name:        "when multiple loaders match, higher index wins",
			packLoaders: []string{"fabric", "forge"},
			fileVers:    []string{"Fabric", "Forge", "1.20.1"},
			wantType:    modloaderTypeFabric,
			wantOK:      true,
		},
		{
			// NeoForge (6) > Quilt (5) > Fabric (4).
			name:        "neoforge beats quilt and fabric when all match",
			packLoaders: []string{"fabric", "quilt", "neoforge"},
			fileVers:    []string{"Fabric", "Quilt", "NeoForge", "1.20.1"},
			wantType:    modloaderTypeNeoForge,
			wantOK:      true,
		},
		{
			name:        "no game versions of any kind means no match",
			packLoaders: []string{"fabric"},
			fileVers:    []string{"1.20.1"},
			wantType:    modloaderTypeAny,
			wantOK:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := modFileInfo{GameVersions: tc.fileVers}

			gotType, gotOK := filterFileInfoLoaderIndex(tc.packLoaders, info)

			if gotType != tc.wantType || gotOK != tc.wantOK {
				t.Errorf("got (%d, %v), want (%d, %v)", gotType, gotOK, tc.wantType, tc.wantOK)
			}
		})
	}
}

func TestMapDepOverride(t *testing.T) {
	const (
		fapiID  uint32 = 306612
		qfapiID uint32 = 634179
		flkID   uint32 = 308769
		qklID   uint32 = 720410
	)

	cases := []struct {
		name      string
		depID     uint32
		isQuilt   bool
		mcVersion string
		want      uint32
	}{
		{"FAPI under Quilt → QFAPI", fapiID, true, "1.20.1", qfapiID},
		{"FAPI under Fabric is left alone", fapiID, false, "1.20.1", fapiID},
		{"FLK under Quilt on 1.20.1 → QKL", flkID, true, "1.20.1", qklID},
		{"FLK under Quilt on 1.19.0 is left alone (below 1.19.2)", flkID, true, "1.19.0", flkID},
		{"FLK under Quilt on 1.18.2 is left alone (below 1.19.2)", flkID, true, "1.18.2", flkID},
		{"unrelated dep ID is left alone", 12345, true, "1.20.1", 12345},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapDepOverride(tc.depID, tc.isQuilt, tc.mcVersion); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestParseExportData(t *testing.T) {
	t.Run("round-trip via ToMap → parseExportData", func(t *testing.T) {
		original := cfExportData{ProjectID: 12345}

		m, err := original.ToMap()
		if err != nil {
			t.Fatalf("ToMap: %v", err)
		}

		back, err := parseExportData(m)
		if err != nil {
			t.Fatalf("parseExportData: %v", err)
		}

		if back != original {
			t.Errorf("round-trip mismatch: got %+v, want %+v", back, original)
		}
	})

	t.Run("missing project-id yields zero value", func(t *testing.T) {
		got, err := parseExportData(map[string]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.ProjectID != 0 {
			t.Errorf("got %+v, want zero ProjectID", got)
		}
	})
}
