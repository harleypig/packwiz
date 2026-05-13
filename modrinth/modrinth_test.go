package modrinth

import (
	"testing"
	"time"

	modrinthApi "codeberg.org/jmansfield/go-modrinth/modrinth"
	"github.com/spf13/viper"
)

func sptr(s string) *string { return &s }

func TestShouldDownloadOnSide(t *testing.T) {
	cases := []struct {
		side string
		want bool
	}{
		{"required", true},
		{"optional", true},
		{"unsupported", false},
		{"", false},
		{"unknown-value", false},
	}

	for _, tc := range cases {
		t.Run(tc.side, func(t *testing.T) {
			if got := shouldDownloadOnSide(tc.side); got != tc.want {
				t.Errorf("shouldDownloadOnSide(%q) = %v, want %v", tc.side, got, tc.want)
			}
		})
	}
}

func TestGetSide(t *testing.T) {
	cases := []struct {
		name       string
		serverSide string
		clientSide string
		want       string
	}{
		{"required on both", "required", "required", "both"},
		{"optional on both", "optional", "optional", "both"},
		{"server only", "required", "unsupported", "server"},
		{"client only", "unsupported", "required", "client"},
		{"unsupported on both", "unsupported", "unsupported", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := &modrinthApi.Project{
				ServerSide: sptr(tc.serverSide),
				ClientSide: sptr(tc.clientSide),
			}

			if got := getSide(project); got != tc.want {
				t.Errorf("getSide = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetBestHash(t *testing.T) {
	t.Run("sha512 wins over weaker hashes", func(t *testing.T) {
		f := &modrinthApi.File{Hashes: map[string]string{
			"sha512": "z", "sha256": "y", "sha1": "x", "murmur2": "w",
		}}

		gotFmt, gotVal := getBestHash(f)
		if gotFmt != "sha512" || gotVal != "z" {
			t.Errorf("got (%q, %q), want (sha512, z)", gotFmt, gotVal)
		}
	})

	t.Run("sha256 picked when sha512 absent", func(t *testing.T) {
		f := &modrinthApi.File{Hashes: map[string]string{"sha256": "v", "sha1": "u"}}

		gotFmt, gotVal := getBestHash(f)
		if gotFmt != "sha256" || gotVal != "v" {
			t.Errorf("got (%q, %q), want (sha256, v)", gotFmt, gotVal)
		}
	})

	t.Run("sha1 picked when only sha1 and murmur2 present", func(t *testing.T) {
		f := &modrinthApi.File{Hashes: map[string]string{"sha1": "s", "murmur2": "m"}}

		gotFmt, gotVal := getBestHash(f)
		if gotFmt != "sha1" || gotVal != "s" {
			t.Errorf("got (%q, %q), want (sha1, s)", gotFmt, gotVal)
		}
	})

	t.Run("murmur2 picked when preferred hashes absent", func(t *testing.T) {
		f := &modrinthApi.File{Hashes: map[string]string{"murmur2": "m"}}

		gotFmt, gotVal := getBestHash(f)
		if gotFmt != "murmur2" || gotVal != "m" {
			t.Errorf("got (%q, %q), want (murmur2, m)", gotFmt, gotVal)
		}
	})

	t.Run("empty hashes return empty strings", func(t *testing.T) {
		f := &modrinthApi.File{Hashes: map[string]string{}}

		gotFmt, gotVal := getBestHash(f)
		if gotFmt != "" || gotVal != "" {
			t.Errorf("got (%q, %q), want (\"\", \"\")", gotFmt, gotVal)
		}
	})
}

func TestCompareLoaderLists(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want int32
	}{
		{
			name: "both empty",
			a:    nil, b: nil,
			want: 0,
		},
		{
			name: "quilt beats fabric",
			a:    []string{"quilt"}, b: []string{"fabric"},
			want: -1,
		},
		{
			name: "fabric loses to quilt",
			a:    []string{"fabric"}, b: []string{"quilt"},
			want: 1,
		},
		{
			name: "neoforge beats forge",
			a:    []string{"neoforge"}, b: []string{"forge"},
			want: -1,
		},
		{
			// Both lists contain "fabric"; compat group makes "quilt"
			// irrelevant for the comparison, so A and B compare equal.
			name: "quilt+fabric vs fabric -> equal via compat group",
			a:    []string{"quilt", "fabric"}, b: []string{"fabric"},
			want: 0,
		},
		{
			// Both lists contain "forge"; "neoforge" enters the compat
			// group and is ignored, leaving forge on both sides.
			name: "forge vs forge+neoforge -> equal via compat group",
			a:    []string{"forge"}, b: []string{"forge", "neoforge"},
			want: 0,
		},
		{
			// PR #391 territory: a multi-loader version that includes
			// fabric beats a neoforge-only version, even when the
			// consumer pack is neoforge-only — because this function
			// has no view of the pack's loaders. Pinning current
			// behavior; the fix (in upstream PR #391) is in the
			// CALLER, not here.
			name: "PR #391 baseline: fabric-bearing list wins regardless of pack loaders",
			a:    []string{"fabric", "forge", "neoforge"}, b: []string{"neoforge"},
			want: -1,
		},
		{
			name: "non-empty vs empty -> non-empty wins",
			a:    []string{"fabric"}, b: nil,
			want: -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compareLoaderLists(tc.a, tc.b); got != tc.want {
				t.Errorf("compareLoaderLists(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestGetProjectTypeFolder(t *testing.T) {
	// Ensure viper datapack-folder is unset for this test's lifetime;
	// none of the cases below need it, and a leak from another test
	// could mask the "datapack" branch behavior.
	t.Cleanup(func() { viper.Set("datapack-folder", "") })
	viper.Set("datapack-folder", "")

	cases := []struct {
		name         string
		projectType  string
		fileLoaders  []string
		packLoaders  []string
		want         string
		wantErr      bool
	}{
		{
			name:        "modpack always errors",
			projectType: "modpack",
			wantErr:     true,
		},
		{
			name:        "resourcepack always returns resourcepacks",
			projectType: "resourcepack",
			want:        "resourcepacks",
		},
		{
			name:        "shader with iris loader goes to shaderpacks",
			projectType: "shader",
			fileLoaders: []string{"iris"},
			want:        "shaderpacks",
		},
		{
			// canvas is in loaderFolders as "resourcepacks" (core
			// shaders ship as resource packs); the function honors
			// that mapping for shader projects too.
			name:        "shader with canvas loader goes to resourcepacks",
			projectType: "shader",
			fileLoaders: []string{"canvas"},
			want:        "resourcepacks",
		},
		{
			name:        "shader with unknown loader falls back to shaderpacks",
			projectType: "shader",
			fileLoaders: []string{"unknown-loader"},
			want:        "shaderpacks",
		},
		{
			name:        "mod with overlapping fabric loader goes to mods",
			projectType: "mod",
			fileLoaders: []string{"fabric"},
			packLoaders: []string{"fabric"},
			want:        "mods",
		},
		{
			name:        "mod with no overlap falls back to mods",
			projectType: "mod",
			fileLoaders: []string{"fabric"},
			packLoaders: []string{"forge"},
			want:        "mods",
		},
		{
			name:        "unknown project type errors",
			projectType: "data-pack",
			wantErr:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getProjectTypeFolder(tc.projectType, tc.fileLoaders, tc.packLoaders)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for project type %q, got nil (result: %q)", tc.projectType, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseSlugOrUrl(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		wantSlug      string
		wantVersion   string
		wantVersionID string
		wantFilename  string
		wantParsed    bool
		wantErr       bool
	}{
		{
			name:     "modrinth URL with project category and slug",
			input:    "https://modrinth.com/mod/sodium",
			wantSlug: "sodium",
		},
		{
			name:        "modrinth URL with slug and version",
			input:       "https://modrinth.com/mod/sodium/version/mc1.20.1",
			wantSlug:    "sodium",
			wantVersion: "mc1.20.1",
		},
		{
			name:          "CDN URL extracts id + versionID + filename",
			input:         "https://cdn.modrinth.com/data/abc123/versions/def456/sodium.jar",
			wantSlug:      "abc123",
			wantVersionID: "def456",
			wantFilename:  "sodium.jar",
		},
		{
			name:    "modrinth URL with unknown category returns error",
			input:   "https://modrinth.com/unknowncategory/sodium",
			wantErr: true,
		},
		{
			name:       "plain slug sets parsedSlug=true",
			input:      "sodium",
			wantSlug:   "sodium",
			wantParsed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var slug, version, versionID, filename string
			parsed, err := parseSlugOrUrl(tc.input, &slug, &version, &versionID, &filename)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if slug != tc.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tc.wantSlug)
			}

			if version != tc.wantVersion {
				t.Errorf("version = %q, want %q", version, tc.wantVersion)
			}

			if versionID != tc.wantVersionID {
				t.Errorf("versionID = %q, want %q", versionID, tc.wantVersionID)
			}

			if filename != tc.wantFilename {
				t.Errorf("filename = %q, want %q", filename, tc.wantFilename)
			}

			if parsed != tc.wantParsed {
				t.Errorf("parsedSlug = %v, want %v", parsed, tc.wantParsed)
			}
		})
	}
}

func TestMapDepOverride(t *testing.T) {
	cases := []struct {
		name      string
		depID     string
		isQuilt   bool
		mcVersion string
		want      string
	}{
		{
			name:      "FAPI ID under Quilt maps to QFAPI/QSL",
			depID:     "P7dR8mSH",
			isQuilt:   true,
			mcVersion: "1.20.1",
			want:      "qvIfYCYJ",
		},
		{
			name:      "FAPI by slug under Quilt also maps",
			depID:     "fabric-api",
			isQuilt:   true,
			mcVersion: "1.20.1",
			want:      "qvIfYCYJ",
		},
		{
			name:      "FAPI under Fabric (not Quilt) is left alone",
			depID:     "P7dR8mSH",
			isQuilt:   false,
			mcVersion: "1.20.1",
			want:      "P7dR8mSH",
		},
		{
			name:      "FLK under Quilt on 1.20.1 maps to QKL",
			depID:     "Ha28R6CL",
			isQuilt:   true,
			mcVersion: "1.20.1",
			want:      "lwVhp9o5",
		},
		{
			name:      "FLK under Quilt on 1.18.2 is left alone (below 1.19.2)",
			depID:     "Ha28R6CL",
			isQuilt:   true,
			mcVersion: "1.18.2",
			want:      "Ha28R6CL",
		},
		{
			name:      "unrelated dep ID is left alone",
			depID:     "abcdef",
			isQuilt:   true,
			mcVersion: "1.20.1",
			want:      "abcdef",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapDepOverride(tc.depID, tc.isQuilt, tc.mcVersion); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// mrVersion is a small builder for *modrinthApi.Version fixtures used
// by the findLatestVersion tests. Each call allocates fresh backing
// strings/times so independent fixtures don't alias one another.
func mrVersion(versionNumber string, gameVersions, loaders []string, date time.Time) *modrinthApi.Version {
	vn := versionNumber
	d := date

	return &modrinthApi.Version{
		VersionNumber: &vn,
		GameVersions:  gameVersions,
		Loaders:       loaders,
		DatePublished: &d,
	}
}

func TestFindLatestVersion_SingleVersionPassesThrough(t *testing.T) {
	only := mrVersion("1.0.0", []string{"1.20.1"}, []string{"fabric"}, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	got := findLatestVersion([]*modrinthApi.Version{only}, []string{"1.20.1"}, false)
	if got != only {
		t.Errorf("expected the only version to be returned, got a different pointer")
	}
}

func TestFindLatestVersion_LaterDateWinsWhenOthersEqual(t *testing.T) {
	// Same version number, game version, loaders — only DatePublished
	// differs. Newer date should win regardless of ordering.
	older := mrVersion("1.0.0", []string{"1.20.1"}, []string{"fabric"}, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	newer := mrVersion("1.0.0", []string{"1.20.1"}, []string{"fabric"}, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))

	t.Run("older first", func(t *testing.T) {
		got := findLatestVersion([]*modrinthApi.Version{older, newer}, []string{"1.20.1"}, false)
		if got != newer {
			t.Errorf("expected newer version, got %q", *got.VersionNumber)
		}
	})

	t.Run("newer first", func(t *testing.T) {
		got := findLatestVersion([]*modrinthApi.Version{newer, older}, []string{"1.20.1"}, false)
		if got != newer {
			t.Errorf("expected newer version, got %q", *got.VersionNumber)
		}
	})
}

func TestFindLatestVersion_HigherGameVersionIndexWins(t *testing.T) {
	// packVersions list semantics: later entries are preferred (the
	// "main" MC version comes last). A version that targets the later
	// entry should beat one that only targets the earlier entry, even
	// when published earlier.
	packVersions := []string{"1.19.4", "1.20.1"}

	olderButNewerMC := mrVersion("1.0.0", []string{"1.20.1"}, []string{"fabric"}, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	newerButOlderMC := mrVersion("2.0.0", []string{"1.19.4"}, []string{"fabric"}, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))

	got := findLatestVersion([]*modrinthApi.Version{newerButOlderMC, olderButNewerMC}, packVersions, false)
	if got != olderButNewerMC {
		t.Errorf("expected the 1.20.1-targeting version to win on game-version index; got %q with MCs %v",
			*got.VersionNumber, got.GameVersions)
	}
}

func TestFindLatestVersion_FlexVerOverridesDate(t *testing.T) {
	// With useFlexVer=true, the version-number compare runs first.
	// A newer flexver beats an older flexver regardless of publish date.
	oldFlexVerNewDate := mrVersion("1.0.0", []string{"1.20.1"}, []string{"fabric"}, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	newFlexVerOldDate := mrVersion("2.0.0", []string{"1.20.1"}, []string{"fabric"}, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	got := findLatestVersion([]*modrinthApi.Version{oldFlexVerNewDate, newFlexVerOldDate}, []string{"1.20.1"}, true)
	if got != newFlexVerOldDate {
		t.Errorf("expected newer-flexver version to win when useFlexVer=true; got %q",
			*got.VersionNumber)
	}
}

func TestFindLatestVersion_LoaderPreference_QuiltOverFabric(t *testing.T) {
	// Identical date and game version; quilt should win over fabric
	// per loaderPreferenceList.
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	fabricOnly := mrVersion("1.0.0", []string{"1.20.1"}, []string{"fabric"}, date)
	quiltOnly := mrVersion("1.0.0", []string{"1.20.1"}, []string{"quilt"}, date)

	got := findLatestVersion([]*modrinthApi.Version{fabricOnly, quiltOnly}, []string{"1.20.1"}, false)
	if got != quiltOnly {
		t.Errorf("expected quilt version to win loader preference; got %v", got.Loaders)
	}
}

func TestFindLatestVersion_PR391Baseline_LoaderListCompareIsPackUnaware(t *testing.T) {
	// PR #391 baseline: findLatestVersion calls compareLoaderLists
	// without filtering each version's loader list to only those
	// relevant to the consumer pack. So a multi-loader version
	// that includes "fabric" can beat a "neoforge"-only version
	// even when the pack is neoforge-only — because fabric ranks
	// ahead of neoforge in loaderPreferenceList.
	//
	// The fix proposed in upstream PR #391 filters each version's
	// loader list to only pack-relevant loaders before comparison.
	// When that lands, this test breaks and the behavior we're
	// pinning will need to be revisited. See .claude/TODO.md for
	// the matched companion task on the CurseForge side.
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	multiLoader := mrVersion("5.0.2", []string{"1.20.1"}, []string{"fabric", "forge", "neoforge"}, date)
	neoforgeOnly := mrVersion("5.4.6.1", []string{"1.20.1"}, []string{"neoforge"}, date)

	// useFlexVer=false so version-number doesn't break the tie.
	got := findLatestVersion([]*modrinthApi.Version{multiLoader, neoforgeOnly}, []string{"1.20.1"}, false)
	if got != multiLoader {
		t.Errorf("baseline pin violated: expected multiLoader (fabric-bearing) to win; got %q", *got.VersionNumber)
	}
}
