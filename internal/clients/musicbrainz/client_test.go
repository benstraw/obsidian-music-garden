package musicbrainz

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSearchArtistByName_usesUserAgentAndCache(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if got := r.Header.Get("User-Agent"); got != "music-garden-test/1.0" {
			t.Fatalf("User-Agent = %q", got)
		}
		if r.URL.Path != "/artist" {
			t.Fatalf("Path = %q", r.URL.Path)
		}
		if q := r.URL.Query().Get("query"); q == "" {
			t.Fatal("expected search query")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":[{"id":"mbid-1","name":"Biosphere","score":"100"}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		UserAgent: "music-garden-test/1.0",
		BaseURL:   server.URL,
		CacheDir:  t.TempDir(),
		CacheTTL:  time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	first, err := client.SearchArtistByName("Biosphere")
	if err != nil {
		t.Fatalf("SearchArtistByName first: %v", err)
	}
	if first.FromCache {
		t.Fatal("first request should not come from cache")
	}
	if first.Value[0].Score != Score(100) {
		t.Fatalf("Score = %v", first.Value[0].Score)
	}
	second, err := client.SearchArtistByName("Biosphere")
	if err != nil {
		t.Fatalf("SearchArtistByName second: %v", err)
	}
	if !second.FromCache {
		t.Fatal("second request should come from cache")
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestGetReleaseGroupByID_parsesLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/release-group/rg-1" {
			t.Fatalf("Path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("inc"); got != "artists+genres+tags+releases" {
			t.Fatalf("inc = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"rg-1",
			"title":"Moon Safari",
			"artist-credit":[{"artist":{"id":"artist-1","name":"Air"}}],
			"genres":[{"name":"downtempo","count":5}],
			"releases":[{"id":"release-1","title":"Moon Safari"}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		UserAgent: "music-garden-test/1.0",
		BaseURL:   server.URL,
		CacheDir:  filepath.Join(t.TempDir(), "cache"),
		CacheTTL:  time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.GetReleaseGroupByID("rg-1")
	if err != nil {
		t.Fatalf("GetReleaseGroupByID: %v", err)
	}
	if result.Value.ID != "rg-1" || result.Value.Title != "Moon Safari" {
		t.Fatalf("unexpected result: %+v", result.Value)
	}
	if got := PrimaryArtistName(result.Value.ArtistCredit); got != "Air" {
		t.Fatalf("PrimaryArtistName = %q", got)
	}
}

func TestEscapeQuery_quotesValues(t *testing.T) {
	got := escapeQuery(`Miles "Tails" Davis`)
	if got != `Miles \"Tails\" Davis` {
		t.Fatalf("escapeQuery = %q", got)
	}
}
