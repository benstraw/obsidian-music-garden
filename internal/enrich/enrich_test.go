package enrich

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mwclient "github.com/benstraw/music-garden/internal/clients/mediawiki"
	"github.com/benstraw/music-garden/internal/genres"
	mwnormalize "github.com/benstraw/music-garden/internal/normalize/mediawiki"
)

func TestIsLikelyGenreEditorialImage(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{title: "File:Ambient_music.jpg", want: true},
		{title: "File:Padlock.svg", want: false},
		{title: "File:Wikipedia-logo-v2.svg", want: false},
	}
	for _, tc := range cases {
		if got := IsLikelyGenreEditorialImage(tc.title); got != tc.want {
			t.Fatalf("IsLikelyGenreEditorialImage(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
}

func TestChooseWikipediaPageTitle_musicQualifiedInCandidates(t *testing.T) {
	title, err := ChooseWikipediaPageTitle(mwnormalize.GenreSeed{CanonicalSlug: "classical", SearchTerm: "classical"}, []mwclient.SearchResult{
		{Title: "Classical"},
		{Title: "Classical music"},
		{Title: "Classical element"},
	})
	if err != nil {
		t.Fatalf("ChooseWikipediaPageTitle: %v", err)
	}
	if title != "Classical music" {
		t.Fatalf("title = %q, want %q", title, "Classical music")
	}
}

func TestChooseWikipediaPageTitle_musicQualifiedNotPresent(t *testing.T) {
	_, err := ChooseWikipediaPageTitle(mwnormalize.GenreSeed{CanonicalSlug: "classical", SearchTerm: "classical"}, []mwclient.SearchResult{
		{Title: "Classical"},
		{Title: "Classical conditioning"},
	})
	if err == nil {
		t.Fatal("ChooseWikipediaPageTitle error = nil, want non-nil")
	}
}

func TestChooseWikipediaPageTitle_explicitPageTitleUnchanged(t *testing.T) {
	title, err := ChooseWikipediaPageTitle(mwnormalize.GenreSeed{CanonicalSlug: "classical", SearchTerm: "classical", PageTitle: "Classical music"}, []mwclient.SearchResult{
		{Title: "Classical music"},
		{Title: "Classical"},
	})
	if err != nil {
		t.Fatalf("ChooseWikipediaPageTitle: %v", err)
	}
	if title != "Classical music" {
		t.Fatalf("title = %q, want %q", title, "Classical music")
	}
}

func TestWikipediaGenre_ambiguousSearchResolvesOnMusicFallback(t *testing.T) {
	searchStems := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/w/api.php"):
			if r.URL.Query().Get("action") == "opensearch" {
				switch r.URL.Query().Get("search") {
				case "classical":
					_, _ = w.Write([]byte(`["classical",["Classical","Classical conditioning","Classical element"],["","",""],["","",""]]`))
					return
				case "classical music":
					_, _ = w.Write([]byte(`["classical music",["Classical music"],["genre page"],["https://en.wikipedia.org/wiki/Classical_music"]]`))
					return
				}
			}
			if r.URL.Query().Get("prop") == "images" {
				_, _ = w.Write([]byte(`{"query":{"pages":{"1":{"images":[]}}}}`))
				return
			}
		case strings.HasPrefix(r.URL.Path, "/api/rest_v1/page/summary/"):
			if strings.Contains(r.URL.Path, "Classical music") || strings.Contains(r.URL.Path, "Classical%20music") {
				_, _ = w.Write([]byte(`{"title":"Classical music","type":"standard","extract":"Classical music is art music.","content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Classical_music"}}}`))
				return
			}
		}
		t.Fatalf("unexpected request: %s", r.URL.String())
	}))
	defer server.Close()

	client, err := mwclient.NewClient(mwclient.Config{
		UserAgent:         "music-garden-test/1.0",
		WikipediaAPIBase:  server.URL + "/w",
		WikipediaRESTBase: server.URL + "/api/rest_v1",
		CommonsAPIBase:    server.URL + "/w",
		CacheDir:          t.TempDir(),
		CacheTTL:          time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	root := t.TempDir()
	store := genres.NewStore()
	status, title, err := WikipediaGenre(root, client, store, mwnormalize.GenreSeed{
		CanonicalSlug: "classical",
		SearchTerm:    "classical",
	}, func(kind, stem, endpoint, requestURL string, fetchedAt time.Time, body []byte, fromCache bool) {
		if kind == "search" {
			searchStems[stem] = true
		}
	}, func(string, ...any) {})
	if err != nil {
		t.Fatalf("WikipediaGenre: %v", err)
	}
	if status != "matched" || title != "Classical music" {
		t.Fatalf("WikipediaGenre = (%q, %q), want (%q, %q)", status, title, "matched", "Classical music")
	}
	if !searchStems["classical"] || !searchStems["classical--music"] {
		t.Fatalf("search stems = %+v", searchStems)
	}
	record, ok := genres.GenreEditorial(store, "classical")
	if !ok || record.WikipediaTitle != "Classical music" {
		t.Fatalf("GenreEditorial = %+v, ok=%v", record, ok)
	}
}

func TestWikipediaGenre_disambiguationPageResolvesOnMusicFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/w/api.php"):
			if r.URL.Query().Get("action") == "opensearch" {
				switch r.URL.Query().Get("search") {
				case "rock":
					_, _ = w.Write([]byte(`["rock",["Rock"],[""],["https://en.wikipedia.org/wiki/Rock"]]`))
					return
				case "rock music":
					_, _ = w.Write([]byte(`["rock music",["Rock music"],["genre page"],["https://en.wikipedia.org/wiki/Rock_music"]]`))
					return
				}
			}
			if r.URL.Query().Get("prop") == "images" {
				_, _ = w.Write([]byte(`{"query":{"pages":{"1":{"images":[]}}}}`))
				return
			}
		case strings.HasPrefix(r.URL.Path, "/api/rest_v1/page/summary/"):
			switch {
			case strings.HasSuffix(r.URL.Path, "/Rock"):
				_, _ = w.Write([]byte(`{"title":"Rock","type":"disambiguation","extract":"Rock may refer to."}`))
				return
			case strings.Contains(r.URL.Path, "Rock music") || strings.Contains(r.URL.Path, "Rock%20music"):
				_, _ = w.Write([]byte(`{"title":"Rock music","type":"standard","extract":"Rock music is a broad genre of popular music.","content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Rock_music"}}}`))
				return
			}
		}
		t.Fatalf("unexpected request: %s", r.URL.String())
	}))
	defer server.Close()

	client, err := mwclient.NewClient(mwclient.Config{
		UserAgent:         "music-garden-test/1.0",
		WikipediaAPIBase:  server.URL + "/w",
		WikipediaRESTBase: server.URL + "/api/rest_v1",
		CommonsAPIBase:    server.URL + "/w",
		CacheDir:          t.TempDir(),
		CacheTTL:          time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	status, title, err := WikipediaGenre(t.TempDir(), client, genres.NewStore(), mwnormalize.GenreSeed{
		CanonicalSlug: "rock",
		SearchTerm:    "rock",
	}, func(kind, stem, endpoint, requestURL string, fetchedAt time.Time, body []byte, fromCache bool) {}, func(string, ...any) {})
	if err != nil {
		t.Fatalf("WikipediaGenre: %v", err)
	}
	if status != "matched" || title != "Rock music" {
		t.Fatalf("WikipediaGenre = (%q, %q), want (%q, %q)", status, title, "matched", "Rock music")
	}
}
