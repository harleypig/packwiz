package cmdshared

import (
	"testing"
	"time"
)

type versionEntry = struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
}

func TestMcVersionManifestCheckValid_Success(t *testing.T) {
	m := McVersionManifest{
		Versions: []versionEntry{
			{ID: "1.19.4"},
			{ID: "1.20.1"},
			{ID: "1.20.2"},
		},
	}

	// Each of these is present in the manifest; CheckValid should
	// return nil. A regression would cause a test failure.
	for _, version := range []string{"1.19.4", "1.20.1", "1.20.2"} {
		t.Run(version, func(t *testing.T) {
			if err := m.CheckValid(version); err != nil {
				t.Errorf("unexpected error for known version %q: %v", version, err)
			}
		})
	}
}

func TestMcVersionManifestCheckValid_Failure(t *testing.T) {
	m := McVersionManifest{
		Versions: []versionEntry{
			{ID: "1.19.4"},
			{ID: "1.20.1"},
		},
	}

	for _, version := range []string{"1.99.0", "", "not-a-version"} {
		t.Run(version, func(t *testing.T) {
			if err := m.CheckValid(version); err == nil {
				t.Errorf("expected error for unknown version %q, got nil", version)
			}
		})
	}
}
