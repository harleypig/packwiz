package curseforge

import (
	"testing"
)

func TestDecodeDefaultKey(t *testing.T) {
	// The default API key is a hardcoded base64 string at the top of
	// request.go. decodeDefaultKey should produce a non-empty result;
	// failure would panic (caught by the test runner).
	got := decodeDefaultKey()

	if got == "" {
		t.Fatal("decodeDefaultKey returned empty string")
	}

	// The decoded value should not still look like base64 — sanity
	// check that decoding actually happened.
	if got == cfApiKeyDefault {
		t.Errorf("decoded value equals the base64 input, decoding likely failed")
	}
}

func TestModFileInfoGetBestHash(t *testing.T) {
	mkHash := func(algo hashAlgo, value string) struct {
		Value     string   `json:"value"`
		Algorithm hashAlgo `json:"algo"`
	} {
		return struct {
			Value     string   `json:"value"`
			Algorithm hashAlgo `json:"algo"`
		}{Value: value, Algorithm: algo}
	}

	type hashEntry = struct {
		Value     string   `json:"value"`
		Algorithm hashAlgo `json:"algo"`
	}

	t.Run("falls back to murmur2 of Fingerprint when Hashes is nil", func(t *testing.T) {
		f := modFileInfo{Fingerprint: 4242}

		hash, format := f.getBestHash()

		if hash != "4242" || format != "murmur2" {
			t.Errorf("got (%q, %q), want (4242, murmur2)", hash, format)
		}
	})

	t.Run("falls back to murmur2 when Hashes is empty", func(t *testing.T) {
		f := modFileInfo{Fingerprint: 7, Hashes: []hashEntry{}}

		hash, format := f.getBestHash()

		if hash != "7" || format != "murmur2" {
			t.Errorf("got (%q, %q), want (7, murmur2)", hash, format)
		}
	})

	t.Run("md5 wins over murmur2", func(t *testing.T) {
		f := modFileInfo{
			Fingerprint: 4242,
			Hashes:      []hashEntry{mkHash(hashAlgoMD5, "md5-value")},
		}

		hash, format := f.getBestHash()

		if hash != "md5-value" || format != "md5" {
			t.Errorf("got (%q, %q), want (md5-value, md5)", hash, format)
		}
	})

	t.Run("sha1 wins over md5 and murmur2", func(t *testing.T) {
		f := modFileInfo{
			Fingerprint: 4242,
			Hashes: []hashEntry{
				mkHash(hashAlgoMD5, "md5-value"),
				mkHash(hashAlgoSHA1, "sha1-value"),
			},
		}

		hash, format := f.getBestHash()

		if hash != "sha1-value" || format != "sha1" {
			t.Errorf("got (%q, %q), want (sha1-value, sha1)", hash, format)
		}
	})

	t.Run("sha1 wins regardless of array order", func(t *testing.T) {
		f := modFileInfo{
			Hashes: []hashEntry{
				mkHash(hashAlgoSHA1, "sha1-value"),
				mkHash(hashAlgoMD5, "md5-value"),
			},
		}

		hash, format := f.getBestHash()

		if hash != "sha1-value" || format != "sha1" {
			t.Errorf("got (%q, %q), want (sha1-value, sha1)", hash, format)
		}
	})
}
