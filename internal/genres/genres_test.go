package genres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/benstraw/music-garden/internal/models"
)

func TestLoad_missingFile(t *testing.T) {
	store, err := Load("/nonexistent/genres.json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(store.Artists) != 0 {
		t.Errorf("expected empty store, got %d artists", len(store.Artists))
	}
	if store.Version != currentVersion {
		t.Fatalf("Version = %d, want %d", store.Version, currentVersion)
	}
}

func TestLoad_legacyMapMigratesToStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genres.json")
	if err := os.WriteFile(path, []byte(`{"abc123":{"name":"Radiohead","genres":["alternative rock","art rock"]}}`), 0644); err != nil {
		t.Fatal(err)
	}

	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	slug := store.ArtistSourceIndex["spotify:abc123"]
	record := store.Artists[slug]
	if record.Name != "Radiohead" {
		t.Fatalf("Name = %q, want Radiohead", record.Name)
	}
	if len(record.Genres) != 2 || record.Genres[0] != "alternative-rock" {
		t.Fatalf("Genres = %v", record.Genres)
	}
}

func TestSaveLoad_roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genres.json")

	store := NewStore()
	Update(store, "abc123", "Radiohead", "https://open.spotify.com/artist/abc123", []string{"alternative rock", "art rock"}, nil)

	if err := Save(path, store); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Artists) != 1 {
		t.Fatalf("expected 1 artist, got %d", len(loaded.Artists))
	}
	record := loaded.Artists["radiohead"]
	if record.Name != "Radiohead" {
		t.Errorf("Name = %q, want Radiohead", record.Name)
	}
	if record.SpotifyArtistID != "abc123" {
		t.Errorf("SpotifyArtistID = %q", record.SpotifyArtistID)
	}
	if record.Genres[0] != "alternative-rock" {
		t.Errorf("Genres = %v", record.Genres)
	}
}

func TestSave_indented(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genres.json")

	store := NewStore()
	Update(store, "x", "Test", "", []string{"rock"}, nil)
	if err := Save(path, store); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, _ := os.ReadFile(path)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("saved JSON is invalid: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	store := NewStore()
	record := Update(store, "id1", "Artist One", "https://open.spotify.com/artist/id1", []string{"pop", "dance pop"}, nil)

	if record.Name != "Artist One" {
		t.Errorf("Name = %q, want Artist One", record.Name)
	}
	if record.Slug != "artist-one" {
		t.Errorf("Slug = %q, want artist-one", record.Slug)
	}
	if len(record.Genres) != 2 {
		t.Errorf("Genres len = %d, want 2", len(record.Genres))
	}
	if store.ArtistSourceIndex["spotify:id1"] != "artist-one" {
		t.Fatalf("source index missing canonical slug")
	}
}

func TestResolvePlay_addsCanonicalIDs(t *testing.T) {
	store := NewStore()
	play := ResolvePlay(store, models.Play{
		Source:           "spotify",
		TrackID:          "track1",
		TrackName:        "Song",
		ArtistID:         "artist1",
		ArtistName:       "Artist One",
		ArtistSpotifyURL: "https://open.spotify.com/artist/artist1",
		AlbumID:          "album1",
		AlbumName:        "Album One",
	})

	if play.ArtistSlug != "artist-one" {
		t.Fatalf("ArtistSlug = %q", play.ArtistSlug)
	}
	if play.ReleaseSlug != "artist-one--album-one" {
		t.Fatalf("ReleaseSlug = %q", play.ReleaseSlug)
	}
}

func TestCanonicalizeTopArtist_tracksPendingGenres(t *testing.T) {
	store := NewStore()
	artist := CanonicalizeTopArtist(store, models.TopArtist{
		ID:     "a1",
		Name:   "Artist A",
		Genres: []string{"indie rock", "mysterious microgenre"},
	})

	if len(artist.Genres) != 1 || artist.Genres[0] != "indie-rock" {
		t.Fatalf("Genres = %v", artist.Genres)
	}
	if len(store.PendingGenreAliases) != 1 || store.PendingGenreAliases[0] != "mysterious microgenre" {
		t.Fatalf("PendingGenreAliases = %v", store.PendingGenreAliases)
	}
}

func TestUpdateImages_preservesGenresAndReplacesImages(t *testing.T) {
	store := NewStore()
	Update(store, "id1", "Artist One", "", []string{"ambient", "electronic"}, []models.ArtistImage{{URL: "https://old", Height: 64, Width: 64}})

	newImages := []models.ArtistImage{
		{URL: "https://img-1", Height: 640, Width: 640},
		{URL: "https://img-2", Height: 320, Width: 320},
	}
	UpdateImages(store, "id1", newImages)

	entry := store.Artists["artist-one"]
	if len(entry.Genres) != 2 || entry.Genres[0] != "ambient" {
		t.Fatalf("genres changed unexpectedly: %v", entry.Genres)
	}
	if len(entry.Images) != 2 || entry.Images[0].URL != "https://img-1" {
		t.Fatalf("images = %+v, want replacement images", entry.Images)
	}
}

func TestMissingImagesArtistIDs(t *testing.T) {
	store := NewStore()
	Update(store, "b", "Has images", "", nil, []models.ArtistImage{{URL: "https://img"}})
	Update(store, "a", "Missing images", "", nil, nil)
	Update(store, "c", "Also missing images", "", nil, []models.ArtistImage{})

	got := MissingImagesArtistIDs(store)
	if len(got) != 2 {
		t.Fatalf("expected 2 ids, got %d: %v", len(got), got)
	}
	if got[0] != "a" || got[1] != "c" {
		t.Fatalf("ids = %v, want [a c]", got)
	}
}

func TestGenresForPlays(t *testing.T) {
	store := NewStore()
	Update(store, "a1", "Artist A", "", []string{"rock", "indie rock"}, nil)
	Update(store, "a2", "Artist B", "", []string{"pop"}, nil)
	plays := []models.Play{
		{ArtistID: "a1", ArtistSlug: "artist-a", ArtistName: "Artist A"},
		{ArtistID: "a1", ArtistSlug: "artist-a", ArtistName: "Artist A"},
		{ArtistID: "a2", ArtistSlug: "artist-b", ArtistName: "Artist B"},
		{ArtistID: "a3", ArtistName: "Artist C"},
	}

	result := GenresForPlays(store, plays)
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if len(result["Artist A"]) != 2 {
		t.Errorf("Artist A genres = %v, want 2", result["Artist A"])
	}
	if len(result["Artist B"]) != 1 {
		t.Errorf("Artist B genres = %v, want 1", result["Artist B"])
	}
	if _, ok := result["Artist C"]; ok {
		t.Error("Artist C should not be in result (not cached)")
	}
}

func TestUncachedArtistIDs(t *testing.T) {
	store := NewStore()
	Update(store, "a1", "Cached", "", []string{"rock"}, nil)
	plays := []models.Play{
		{ArtistID: "a1", ArtistName: "Cached"},
		{ArtistID: "a2", ArtistName: "New1"},
		{ArtistID: "a3", ArtistName: "New2"},
		{ArtistID: "a2", ArtistName: "New1"},
		{ArtistID: "", ArtistName: "Empty"},
	}

	ids := UncachedArtistIDs(store, plays)
	if len(ids) != 2 {
		t.Fatalf("expected 2 uncached IDs, got %d: %v", len(ids), ids)
	}
	if ids[0] != "a2" || ids[1] != "a3" {
		t.Errorf("ids = %v, want [a2, a3]", ids)
	}
}
