package mediawiki

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSearchPagesAndSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/w/api.php"):
			if r.URL.Query().Get("action") == "opensearch" {
				_, _ = w.Write([]byte(`["ambient music",["Ambient music"],["genre page"],["https://en.wikipedia.org/wiki/Ambient_music"]]`))
				return
			}
			if r.URL.Query().Get("prop") == "images" {
				_, _ = w.Write([]byte(`{"query":{"pages":{"1":{"images":[{"title":"File:Ambient_music.jpg"}]}}}}`))
				return
			}
		case strings.HasPrefix(r.URL.Path, "/api/rest_v1/page/summary/"):
			_, _ = w.Write([]byte(`{
				"title":"Ambient music",
				"extract":"Ambient music is a genre of music.",
				"content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Ambient_music"}},
				"thumbnail":{"source":"https://upload.wikimedia.org/thumb.jpg","width":320,"height":180}
			}`))
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client, err := NewClient(Config{
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

	search, err := client.SearchPages("ambient music")
	if err != nil {
		t.Fatalf("SearchPages: %v", err)
	}
	if len(search.Value) != 1 || search.Value[0].Title != "Ambient music" {
		t.Fatalf("unexpected search results: %+v", search.Value)
	}

	summary, err := client.GetSummary("Ambient music")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary.Value.Title != "Ambient music" || summary.Value.Extract == "" {
		t.Fatalf("unexpected summary: %+v", summary.Value)
	}

	images, err := client.GetPageImages("Ambient music")
	if err != nil {
		t.Fatalf("GetPageImages: %v", err)
	}
	if len(images.Value) != 1 || images.Value[0] != "File:Ambient_music.jpg" {
		t.Fatalf("unexpected page images: %+v", images.Value)
	}
}

func TestGetCommonsImageInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/w/api.php") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"query":{"pages":{"1":{
				"title":"File:Ambient_music.jpg",
				"imageinfo":[{
					"url":"https://upload.wikimedia.org/original.jpg",
					"descriptionurl":"https://commons.wikimedia.org/wiki/File:Ambient_music.jpg",
					"thumburl":"https://upload.wikimedia.org/thumb.jpg",
					"width":1200,
					"height":800,
					"extmetadata":{
						"LicenseShortName":{"value":"CC BY-SA 4.0"},
						"LicenseUrl":{"value":"https://creativecommons.org/licenses/by-sa/4.0/"},
						"Artist":{"value":"Example Author"},
						"Attribution":{"value":"Example attribution"}
					}
				}]
			}}}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		UserAgent:      "music-garden-test/1.0",
		CommonsAPIBase: server.URL + "/w",
		CacheDir:       t.TempDir(),
		CacheTTL:       time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	info, err := client.GetCommonsImageInfo("File:Ambient_music.jpg")
	if err != nil {
		t.Fatalf("GetCommonsImageInfo: %v", err)
	}
	if info.Value.FileTitle != "File:Ambient_music.jpg" || info.Value.License == "" {
		t.Fatalf("unexpected image info: %+v", info.Value)
	}
}
