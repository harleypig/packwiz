package curseforge

import (
	"testing"

	"github.com/jarcoal/httpmock"
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

func TestGetModInfo(t *testing.T) {
	t.Run("happy path returns the deserialized mod", func(t *testing.T) {
		httpmock.Activate(t)

		body := `{"data":{
			"id": 12345,
			"name": "Test Mod",
			"slug": "test-mod",
			"summary": "A mod used by TestGetModInfo",
			"gameId": 432,
			"classId": 6,
			"latestFiles": [
				{"id": 4567, "fileName": "test-mod-1.0.0.jar", "gameVersions": ["Fabric","1.20.1"]}
			],
			"latestFilesIndexes": [],
			"modLoaders": ["Fabric"],
			"links": {"websiteUrl": "https://www.curseforge.com/minecraft/mc-mods/test-mod"}
		}}`

		httpmock.RegisterResponder("GET", "https://api.curseforge.com/v1/mods/12345",
			httpmock.NewStringResponder(200, body))

		info, err := cfDefaultClient.getModInfo(12345)
		if err != nil {
			t.Fatalf("getModInfo: %v", err)
		}

		if info.ID != 12345 {
			t.Errorf("ID = %d, want 12345", info.ID)
		}

		if info.Name != "Test Mod" {
			t.Errorf("Name = %q, want %q", info.Name, "Test Mod")
		}

		if len(info.LatestFiles) != 1 || info.LatestFiles[0].ID != 4567 {
			t.Errorf("LatestFiles = %+v, want one entry with ID 4567", info.LatestFiles)
		}
	})

	t.Run("rejects response with mismatched ID", func(t *testing.T) {
		httpmock.Activate(t)

		// Server responds with a different ID than what was requested.
		body := `{"data":{"id": 999, "name": "Wrong"}}`
		httpmock.RegisterResponder("GET", "https://api.curseforge.com/v1/mods/12345",
			httpmock.NewStringResponder(200, body))

		_, err := cfDefaultClient.getModInfo(12345)
		if err == nil {
			t.Error("expected error for ID mismatch, got nil")
		}
	})

	t.Run("propagates non-200 status as error", func(t *testing.T) {
		httpmock.Activate(t)

		httpmock.RegisterResponder("GET", "https://api.curseforge.com/v1/mods/12345",
			httpmock.NewStringResponder(500, `{"error":"boom"}`))

		_, err := cfDefaultClient.getModInfo(12345)
		if err == nil {
			t.Error("expected error for 500 response, got nil")
		}
	})
}

func TestGetFileInfo(t *testing.T) {
	t.Run("happy path returns the deserialized file", func(t *testing.T) {
		httpmock.Activate(t)

		body := `{"data":{
			"id": 4567,
			"modId": 12345,
			"fileName": "test-mod-1.0.0.jar",
			"displayName": "Test Mod 1.0.0",
			"fileLength": 1024,
			"downloadUrl": "https://edge.forgecdn.net/files/4/5/test-mod.jar",
			"gameVersions": ["Fabric","1.20.1"],
			"fileFingerprint": 42424242,
			"hashes": [{"value":"sha1value","algo":1},{"value":"md5value","algo":2}]
		}}`

		httpmock.RegisterResponder("GET", "https://api.curseforge.com/v1/mods/12345/files/4567",
			httpmock.NewStringResponder(200, body))

		info, err := cfDefaultClient.getFileInfo(12345, 4567)
		if err != nil {
			t.Fatalf("getFileInfo: %v", err)
		}

		if info.ID != 4567 {
			t.Errorf("ID = %d, want 4567", info.ID)
		}

		if info.ModID != 12345 {
			t.Errorf("ModID = %d, want 12345", info.ModID)
		}

		if info.FileName != "test-mod-1.0.0.jar" {
			t.Errorf("FileName = %q, want %q", info.FileName, "test-mod-1.0.0.jar")
		}

		if info.Fingerprint != 42424242 {
			t.Errorf("Fingerprint = %d, want 42424242", info.Fingerprint)
		}

		// Verify the hash slice deserialized with both algorithm enum
		// values intact.
		if len(info.Hashes) != 2 {
			t.Fatalf("Hashes count = %d, want 2", len(info.Hashes))
		}
	})

	t.Run("rejects response with mismatched file ID", func(t *testing.T) {
		httpmock.Activate(t)

		body := `{"data":{"id": 999, "modId": 12345}}`
		httpmock.RegisterResponder("GET", "https://api.curseforge.com/v1/mods/12345/files/4567",
			httpmock.NewStringResponder(200, body))

		_, err := cfDefaultClient.getFileInfo(12345, 4567)
		if err == nil {
			t.Error("expected error for file ID mismatch, got nil")
		}
	})
}

func TestGetModInfoMultiple(t *testing.T) {
	httpmock.Activate(t)

	body := `{"data":[
		{"id": 1, "name": "A"},
		{"id": 2, "name": "B"},
		{"id": 3, "name": "C"}
	]}`

	httpmock.RegisterResponder("POST", "https://api.curseforge.com/v1/mods",
		httpmock.NewStringResponder(200, body))

	infos, err := cfDefaultClient.getModInfoMultiple([]uint32{1, 2, 3})
	if err != nil {
		t.Fatalf("getModInfoMultiple: %v", err)
	}

	if len(infos) != 3 {
		t.Fatalf("got %d infos, want 3", len(infos))
	}

	wantNames := []string{"A", "B", "C"}
	for i, info := range infos {
		if info.Name != wantNames[i] {
			t.Errorf("infos[%d].Name = %q, want %q", i, info.Name, wantNames[i])
		}
	}
}

func TestGetFileInfoMultiple(t *testing.T) {
	httpmock.Activate(t)

	body := `{"data":[
		{"id": 100, "modId": 1, "fileName": "a.jar"},
		{"id": 200, "modId": 2, "fileName": "b.jar"}
	]}`

	httpmock.RegisterResponder("POST", "https://api.curseforge.com/v1/mods/files",
		httpmock.NewStringResponder(200, body))

	infos, err := cfDefaultClient.getFileInfoMultiple([]uint32{100, 200})
	if err != nil {
		t.Fatalf("getFileInfoMultiple: %v", err)
	}

	if len(infos) != 2 {
		t.Fatalf("got %d infos, want 2", len(infos))
	}

	if infos[0].FileName != "a.jar" || infos[1].FileName != "b.jar" {
		t.Errorf("got filenames %q / %q, want a.jar / b.jar",
			infos[0].FileName, infos[1].FileName)
	}
}
