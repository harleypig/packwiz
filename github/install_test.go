package github

import (
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestGithubRegex(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantSlug  string
		wantMatch bool
	}{
		{
			name:      "plain URL",
			input:     "https://github.com/packwiz/packwiz",
			wantSlug:  "packwiz/packwiz",
			wantMatch: true,
		},
		{
			name:      "URL with www prefix",
			input:     "https://www.github.com/packwiz/packwiz",
			wantSlug:  "packwiz/packwiz",
			wantMatch: true,
		},
		{
			name:      "http (not https)",
			input:     "http://github.com/packwiz/packwiz",
			wantSlug:  "packwiz/packwiz",
			wantMatch: true,
		},
		{
			name:      "extra path segments are ignored after the slug",
			input:     "https://github.com/packwiz/packwiz/releases/tag/v1.0",
			wantSlug:  "packwiz/packwiz",
			wantMatch: true,
		},
		{
			name:      "not a github URL",
			input:     "https://example.com/foo/bar",
			wantMatch: false,
		},
		{
			name:      "plain slug (no protocol) does not match",
			input:     "packwiz/packwiz",
			wantMatch: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches := GithubRegex.FindStringSubmatch(tc.input)

			gotMatch := len(matches) == 2
			if gotMatch != tc.wantMatch {
				t.Fatalf("match = %v, want %v", gotMatch, tc.wantMatch)
			}

			if tc.wantMatch && matches[1] != tc.wantSlug {
				t.Errorf("slug = %q, want %q", matches[1], tc.wantSlug)
			}
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	const slug = "packwiz/packwiz"
	const releasesURL = "https://api.github.com/repos/" + slug + "/releases"

	t.Run("no branch returns the first release in the array", func(t *testing.T) {
		httpmock.Activate(t)

		body := `[
			{"url":"r1", "tag_name":"v2.0.0", "target_commitish":"main", "name":"Two", "created_at":"2024-06-01T00:00:00Z", "assets":[]},
			{"url":"r2", "tag_name":"v1.0.0", "target_commitish":"main", "name":"One", "created_at":"2024-01-01T00:00:00Z", "assets":[]}
		]`

		httpmock.RegisterResponder("GET", releasesURL,
			httpmock.NewStringResponder(200, body))

		release, err := getLatestRelease(slug, "")
		if err != nil {
			t.Fatalf("getLatestRelease: %v", err)
		}

		if release.TagName != "v2.0.0" {
			t.Errorf("TagName = %q, want v2.0.0", release.TagName)
		}
	})

	t.Run("branch filter returns the matching release", func(t *testing.T) {
		httpmock.Activate(t)

		body := `[
			{"tag_name":"v2.0.0-main", "target_commitish":"main"},
			{"tag_name":"v1.5.0-stable", "target_commitish":"stable"},
			{"tag_name":"v1.0.0-stable", "target_commitish":"stable"}
		]`

		httpmock.RegisterResponder("GET", releasesURL,
			httpmock.NewStringResponder(200, body))

		release, err := getLatestRelease(slug, "stable")
		if err != nil {
			t.Fatalf("getLatestRelease: %v", err)
		}

		// The branch loop returns the FIRST match in the array, so
		// v1.5.0-stable wins over v1.0.0-stable.
		if release.TagName != "v1.5.0-stable" {
			t.Errorf("TagName = %q, want v1.5.0-stable", release.TagName)
		}
	})

	t.Run("branch with no match returns error", func(t *testing.T) {
		httpmock.Activate(t)

		body := `[
			{"tag_name":"v1.0.0", "target_commitish":"main"}
		]`

		httpmock.RegisterResponder("GET", releasesURL,
			httpmock.NewStringResponder(200, body))

		_, err := getLatestRelease(slug, "nonexistent-branch")
		if err == nil {
			t.Error("expected error for branch with no matching release, got nil")
		}
	})
}
