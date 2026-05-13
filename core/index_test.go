package core

import (
	"path/filepath"
	"testing"
)

// newTestIndex builds an Index rooted at packRoot with no files.
// Tests can populate Files directly before exercising methods.
func newTestIndex(packRoot string) Index {
	return Index{
		HashFormat: "sha256",
		Files:      IndexFiles{},
		indexFile:  filepath.Join(packRoot, "index.toml"),
		packRoot:   packRoot,
	}
}

func TestResolveIndexPath(t *testing.T) {
	in := newTestIndex(filepath.Join("some", "pack"))

	// Index paths are stored forward-slashed; ResolveIndexPath converts
	// to OS-native and joins with packRoot.
	got := in.ResolveIndexPath("mods/sodium.pw.toml")
	want := filepath.Join("some", "pack", "mods", "sodium.pw.toml")

	if got != want {
		t.Errorf("ResolveIndexPath = %q, want %q", got, want)
	}
}

func TestRelIndexPath(t *testing.T) {
	root := filepath.Join("some", "pack")
	in := newTestIndex(root)

	got, err := in.RelIndexPath(filepath.Join(root, "mods", "sodium.pw.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "mods/sodium.pw.toml" {
		t.Errorf("RelIndexPath = %q, want %q", got, "mods/sodium.pw.toml")
	}
}

func TestFindMod(t *testing.T) {
	root := filepath.Join("some", "pack")
	in := newTestIndex(root)
	in.Files["mods/sodium.pw.toml"] = &indexFile{File: "mods/sodium.pw.toml", MetaFile: true, fileFound: true}
	in.Files["mods/iris.pw.toml"] = &indexFile{File: "mods/iris.pw.toml", MetaFile: true, fileFound: true}
	in.Files["overrides/config.txt"] = &indexFile{File: "overrides/config.txt", fileFound: true}

	t.Run("finds a metafile by slug", func(t *testing.T) {
		got, ok := in.FindMod("sodium")
		if !ok {
			t.Fatal("expected to find sodium")
		}

		want := filepath.Join(root, "mods", "sodium.pw.toml")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("misses unknown slug", func(t *testing.T) {
		if _, ok := in.FindMod("does-not-exist"); ok {
			t.Error("expected miss for unknown slug")
		}
	})

	t.Run("skips non-metafile entries even if name matches", func(t *testing.T) {
		// "config" would match overrides/config.txt's filename minus
		// extension, but it is not a metafile and so should not be
		// returned.
		if _, ok := in.FindMod("config"); ok {
			t.Error("expected miss for non-metafile entry")
		}
	})
}

func TestRemoveFile(t *testing.T) {
	root := filepath.Join("some", "pack")
	in := newTestIndex(root)
	in.Files["mods/sodium.pw.toml"] = &indexFile{File: "mods/sodium.pw.toml"}

	err := in.RemoveFile(filepath.Join(root, "mods", "sodium.pw.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, present := in.Files["mods/sodium.pw.toml"]; present {
		t.Error("entry was not removed")
	}
}

func TestRefreshFileWithHash(t *testing.T) {
	root := filepath.Join("some", "pack")
	in := newTestIndex(root)

	err := in.RefreshFileWithHash(
		filepath.Join(root, "mods", "sodium.pw.toml"),
		"sha256", "abc123", true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	holder, ok := in.Files["mods/sodium.pw.toml"]
	if !ok {
		t.Fatal("entry was not created")
	}

	if !holder.IsMetaFile() {
		t.Error("entry should be marked as metafile")
	}

	f := holder.(*indexFile)
	// updateFileHashGiven blanks the format when it matches the index hash format.
	if f.Hash != "abc123" || f.HashFormat != "" {
		t.Errorf("got hash=%q format=%q, want hash=abc123 format=\"\"", f.Hash, f.HashFormat)
	}
}
