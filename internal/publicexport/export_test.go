package publicexport

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
	"github.com/benstraw/music-garden/internal/reviews"
)

func TestExportDeterministicPrivateAndAtomic(t *testing.T) {
	catalog := exportFixtureCatalog()
	plays := exportFixturePlays()
	reviewStore, err := reviews.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	firstDir := filepath.Join(t.TempDir(), "v1")
	secondDir := filepath.Join(t.TempDir(), "v1")
	for _, out := range []string{firstDir, secondDir} {
		if _, err := Export(Options{OutputDir: out, SourceRevision: "abc123", Timezone: "America/Los_Angeles", Catalog: catalog, Plays: plays, Reviews: reviewStore}); err != nil {
			t.Fatalf("Export(%s): %v", out, err)
		}
	}

	first := readExportTree(t, firstDir)
	second := readExportTree(t, secondDir)
	if len(first) != len(second) {
		t.Fatalf("file counts differ: %d != %d", len(first), len(second))
	}
	for path, data := range first {
		if !bytes.Equal(data, second[path]) {
			t.Errorf("non-deterministic output: %s", path)
		}
		if strings.Contains(string(data), "played_at") || strings.Contains(string(data), "2024-01-01") || strings.Contains(string(data), "synthetic") {
			t.Errorf("private or synthetic data leaked through %s", path)
		}
	}
	if _, ok := first["weeks/2026/2026-W32.json"]; !ok {
		t.Fatalf("expected 2026 weekly shard; files=%v", mapKeys(first))
	}

	stale := filepath.Join(firstDir, "weeks", "2020", "stale.json")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(Options{OutputDir: firstDir, SourceRevision: "abc123", Timezone: "America/Los_Angeles", Catalog: catalog, Plays: plays, Reviews: reviewStore}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale output survived replacement: %v", err)
	}
}

func TestValidateDirectoryRejectsChangedFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "v1")
	manifest, err := Export(Options{OutputDir: out, SourceRevision: "abc123", Timezone: "America/Los_Angeles", Catalog: exportFixtureCatalog(), Plays: exportFixturePlays(), Reviews: mustReviewStore(t)})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(out, manifest.Files[0].Path)
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateDirectory(out, manifest); err == nil {
		t.Fatal("expected checksum validation failure")
	}
}

func TestISOWeekStartHandlesSundayReferenceYear(t *testing.T) {
	loc := time.UTC
	got := isoWeekStart(2027, 1, loc)
	if got.Format("2006-01-02") != "2027-01-04" {
		t.Fatalf("week one starts %s", got.Format("2006-01-02"))
	}
}

func exportFixtureCatalog() *genres.Catalog {
	catalog := genres.NewCatalog()
	catalog.Genres.GenreRecords["ambient"] = genres.GenreRecord{Slug: "ambient", DisplayName: "Ambient", WorkflowState: genres.WorkflowStatePublishable, Summary: "Atmospheric music."}
	for _, artist := range []genres.ArtistRecord{
		{Slug: "alpha", Name: "Alpha", SpotifyArtistID: "a", Genres: []string{"ambient"}},
		{Slug: "beta", Name: "Beta", SpotifyArtistID: "b", Genres: []string{"ambient"}},
	} {
		catalog.Artists.Artists[artist.Slug] = artist
		catalog.Artists.ArtistSourceIndex["spotify:"+artist.SpotifyArtistID] = artist.Slug
	}
	catalog.Releases.Releases["alpha--ordered"] = genres.ReleaseRecord{
		Slug: "alpha--ordered", Name: "Ordered", PrimaryArtistSlug: "alpha", PrimaryArtistName: "Alpha", SpotifyAlbumID: "album",
		Editions: []genres.ReleaseEdition{{SpotifyAlbumID: "album", Name: "Ordered", TotalTracks: 4, Tracks: []genres.ReleaseTrack{{DiscNumber: 1, TrackNumber: 1, Name: "One"}, {DiscNumber: 1, TrackNumber: 2, Name: "Two"}, {DiscNumber: 1, TrackNumber: 3, Name: "Three"}, {DiscNumber: 1, TrackNumber: 4, Name: "Four"}}}},
	}
	catalog.Releases.ReleaseSourceIndex["spotify:album"] = "alpha--ordered"
	return catalog
}

func exportFixturePlays() []models.Play {
	plays := []models.Play{
		fixturePlay("2026-08-03T10:00:00Z", "alpha", "a", "album", "alpha--ordered", "one", 1),
		fixturePlay("2026-08-03T10:04:00Z", "alpha", "a", "album", "alpha--ordered", "two", 2),
		fixturePlay("2026-08-03T10:08:00Z", "alpha", "a", "album", "alpha--ordered", "three", 3),
		fixturePlay("2026-08-04T10:00:00Z", "alpha", "a", "other", "", "extra", 1),
		fixturePlay("2026-08-03T12:00:00Z", "beta", "b", "other", "", "b1", 1),
		fixturePlay("2026-08-03T12:04:00Z", "beta", "b", "other", "", "b2", 2),
		fixturePlay("2026-08-04T12:00:00Z", "beta", "b", "other", "", "b3", 3),
		{PlayedAt: "2024-01-01T12:34:56Z", Source: "synthetic-legacy", TrackID: "fake", ArtistSlug: "alpha", ArtistID: "a", ArtistName: "Alpha"},
	}
	return plays
}

func fixturePlay(at, slug, artistID, albumID, releaseSlug, trackID string, number int) models.Play {
	return models.Play{PlayedAt: at, Source: "spotify", TrackID: trackID, TrackName: strings.ToUpper(trackID), ArtistSlug: slug, ArtistID: artistID, ArtistName: strings.ToUpper(slug), AlbumID: albumID, AlbumName: "Ordered", ReleaseSlug: releaseSlug, AlbumTotalTracks: 4, DiscNumber: 1, TrackNumber: number, DurationMS: 180000}
}

func mustReviewStore(t *testing.T) *reviews.Store {
	t.Helper()
	store, err := reviews.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func readExportTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		data, err := os.ReadFile(path)
		if err == nil {
			result[filepath.ToSlash(rel)] = data
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mapKeys(values map[string][]byte) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}
