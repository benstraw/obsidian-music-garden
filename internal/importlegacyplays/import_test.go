package importlegacyplays

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
)

func TestPrepare_GeneratesDeterministicLegacyBackfillPlays(t *testing.T) {
	sourceDir := t.TempDir()
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[
	  {
	    "id":"t1","name":"Track One","duration_ms":1000,
	    "external_urls":{"spotify":"https://open.spotify.com/track/t1"},
	    "artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}],
	    "album":{"id":"al1","name":"Album One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}]}
	  },
	  {
	    "id":"t2","name":"Track Two","duration_ms":2000,
	    "external_urls":{"spotify":"https://open.spotify.com/track/t2"},
	    "artists":[
	      {"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}},
	      {"id":"a2","name":"Guest Two","external_urls":{"spotify":"https://open.spotify.com/artist/a2"}}
	    ],
	    "album":{"id":"al1","name":"Album One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}]}
	  }
	]}`)
	mustWrite(t, filepath.Join(sourceDir, "artists.json"), `{
	  "artist-one":{"id":"a1","name":"Artist One","spotify_url":"https://open.spotify.com/artist/a1","genres":["hip hop"],"first_seen":"2024-01-10","last_seen":"2024-01-20"}
	}`)

	opts := Options{
		SourceDir:    sourceDir,
		ManifestPath: filepath.Join(t.TempDir(), "legacy-plays.json"),
		FallbackFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		FallbackTo:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	result1, err := Prepare(genres.NewCatalog(), opts)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	result2, err := Prepare(genres.NewCatalog(), Options{
		SourceDir:    sourceDir,
		ManifestPath: filepath.Join(t.TempDir(), "legacy-plays-2.json"),
		FallbackFrom: opts.FallbackFrom,
		FallbackTo:   opts.FallbackTo,
	})
	if err != nil {
		t.Fatalf("Prepare second: %v", err)
	}
	if len(result1.Plays) != 2 {
		t.Fatalf("expected 2 plays, got %d", len(result1.Plays))
	}
	for i := range result1.Plays {
		if result1.Plays[i].Source != "legacy-backfill" {
			t.Fatalf("play %d source = %q", i, result1.Plays[i].Source)
		}
		if result1.Plays[i].PlayedAt != result2.Plays[i].PlayedAt {
			t.Fatalf("non-deterministic played_at at %d: %q vs %q", i, result1.Plays[i].PlayedAt, result2.Plays[i].PlayedAt)
		}
	}
	if got := result1.Plays[1].AdditionalArtists; len(got) != 1 || got[0].Name != "Guest Two" {
		t.Fatalf("unexpected additional artists: %#v", got)
	}
}

func TestPrepare_FallbackWindowUsedWhenLegacyDatesMissing(t *testing.T) {
	sourceDir := t.TempDir()
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[
	  {
	    "id":"t1","name":"Track One","duration_ms":1000,
	    "external_urls":{"spotify":"https://open.spotify.com/track/t1"},
	    "artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}],
	    "album":{"id":"al1","name":"Album One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}]}
	  }
	]}`)
	mustWrite(t, filepath.Join(sourceDir, "artists.json"), `{
	  "artist-one":{"id":"a1","name":"Artist One","spotify_url":"https://open.spotify.com/artist/a1","genres":["hip hop"],"first_seen":"","last_seen":""}
	}`)
	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	result, err := Prepare(genres.NewCatalog(), Options{
		SourceDir:    sourceDir,
		ManifestPath: filepath.Join(t.TempDir(), "legacy-plays.json"),
		FallbackFrom: from,
		FallbackTo:   to,
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if result.Summary.ArtistsFallback != 1 {
		t.Fatalf("expected fallback artist count 1, got %d", result.Summary.ArtistsFallback)
	}
	got, err := time.Parse(time.RFC3339Nano, result.Plays[0].PlayedAt)
	if err != nil {
		t.Fatalf("parse played_at: %v", err)
	}
	if got.Before(from) || got.After(time.Date(2024, 2, 29, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)) {
		t.Fatalf("played_at %s outside fallback window", got)
	}
}

func TestPrepare_TrimsFallbackWindowToISOYear(t *testing.T) {
	sourceDir := t.TempDir()
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[
	  {
	    "id":"t1","name":"Track One","duration_ms":1000,
	    "external_urls":{"spotify":"https://open.spotify.com/track/t1"},
	    "artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}],
	    "album":{"id":"al1","name":"Album One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}}]}
	  }
	]}`)
	mustWrite(t, filepath.Join(sourceDir, "artists.json"), `{
	  "artist-one":{"id":"a1","name":"Artist One","spotify_url":"https://open.spotify.com/artist/a1","genres":["hip hop"],"first_seen":"2024-12-31","last_seen":"2024-12-31"}
	}`)

	result, err := Prepare(genres.NewCatalog(), Options{
		SourceDir:    sourceDir,
		ManifestPath: filepath.Join(t.TempDir(), "legacy-plays.json"),
		FallbackFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		FallbackTo:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	got, err := time.Parse(time.RFC3339Nano, result.Plays[0].PlayedAt)
	if err != nil {
		t.Fatalf("parse played_at: %v", err)
	}
	if got.Year() != 2024 {
		t.Fatalf("played_at year = %d, want 2024", got.Year())
	}
	isoYear, _ := got.ISOWeek()
	if isoYear != 2024 {
		t.Fatalf("played_at ISO year = %d, want 2024", isoYear)
	}
}

func TestPrepare_ManifestHardStop(t *testing.T) {
	sourceDir := t.TempDir()
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[]}`)
	manifestPath := filepath.Join(t.TempDir(), "legacy-plays.json")
	mustWrite(t, manifestPath, `{"prepared_plays":1}`)

	_, err := Prepare(genres.NewCatalog(), Options{
		SourceDir:    sourceDir,
		ManifestPath: manifestPath,
		FallbackFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		FallbackTo:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if err == nil || !strings.Contains(err.Error(), "manifest already exists") {
		t.Fatalf("expected manifest hard stop, got %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
