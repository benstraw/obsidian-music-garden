package genres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/benstraw/music-garden/internal/models"
)

func applyTestTaxonomy(store *Store) {
	ApplyTaxonomy(store, &Taxonomy{
		Version: 1,
		Genres: []GenreDefinition{
			{Slug: "alternative-rock", DisplayName: "Alternative Rock", Aliases: []string{"alternative rock"}},
			{Slug: "art-rock", DisplayName: "Art Rock", Aliases: []string{"art rock"}},
			{Slug: "ambient", DisplayName: "Ambient", Aliases: []string{"ambient"}},
			{Slug: "electronic", DisplayName: "Electronic", Aliases: []string{"electronic"}},
			{Slug: "rock", DisplayName: "Rock", Aliases: []string{"rock"}},
			{Slug: "indie-rock", DisplayName: "Indie Rock", Aliases: []string{"indie rock"}},
			{Slug: "pop", DisplayName: "Pop", Aliases: []string{"pop"}},
			{Slug: "dance-pop", DisplayName: "Dance Pop", Aliases: []string{"dance pop"}},
		},
	})
}

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
	applyTestTaxonomy(store)

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
	genresPath := filepath.Join(dir, "genres.json")
	artistsPath := filepath.Join(dir, "artists.json")
	releasesPath := filepath.Join(dir, "releases.json")

	store := NewStore()
	applyTestTaxonomy(store)
	Update(store, "abc123", "Radiohead", "https://open.spotify.com/artist/abc123", []string{"alternative rock", "art rock"}, nil)

	catalog := NewCatalog()
	catalog.Genres = store
	catalog.Artists.Artists = store.Artists
	catalog.Artists.ArtistSlugAliases = store.ArtistSlugAliases
	catalog.Artists.ArtistSourceIndex = store.ArtistSourceIndex
	catalog.Releases.Releases = store.Releases
	catalog.Releases.ReleaseSourceIndex = store.ReleaseSourceIndex

	if err := SaveCatalog(genresPath, artistsPath, releasesPath, catalog); err != nil {
		t.Fatalf("SaveCatalog: %v", err)
	}

	loaded, err := LoadCatalog(genresPath, artistsPath, releasesPath)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	if len(loaded.Artists.Artists) != 1 {
		t.Fatalf("expected 1 artist, got %d", len(loaded.Artists.Artists))
	}
	record := loaded.Artists.Artists["radiohead"]
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

func TestSaveCatalog_splitsGenreArtistReleaseFiles(t *testing.T) {
	dir := t.TempDir()
	genresPath := filepath.Join(dir, "genres.json")
	artistsPath := filepath.Join(dir, "artists.json")
	releasesPath := filepath.Join(dir, "releases.json")

	catalog := NewCatalog()
	applyTestTaxonomy(catalog.Genres)
	artist := Update(catalog, "artist1", "Artist One", "https://open.spotify.com/artist/artist1", []string{"rock"}, nil)
	_ = UpsertReleaseMetadata(catalog, artist, "album1", "Album One", "rg1", "rel1")

	if err := SaveCatalog(genresPath, artistsPath, releasesPath, catalog); err != nil {
		t.Fatalf("SaveCatalog: %v", err)
	}

	genreData, err := os.ReadFile(genresPath)
	if err != nil {
		t.Fatalf("ReadFile genresPath: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(genreData, &raw); err != nil {
		t.Fatalf("saved genres JSON is invalid: %v", err)
	}
	if raw["artists"] != nil || raw["releases"] != nil {
		t.Fatalf("genres.json still contains entity sections")
	}

	if _, err := os.Stat(artistsPath); err != nil {
		t.Fatalf("artists.json missing: %v", err)
	}
	if _, err := os.Stat(releasesPath); err != nil {
		t.Fatalf("releases.json missing: %v", err)
	}
}

func TestSave_indented(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genres.json")

	store := NewStore()
	applyTestTaxonomy(store)
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

func TestGenreWorkflowState_defaultsToDraft(t *testing.T) {
	record := GenreRecord{Slug: "indie-rock"}
	if got := GenreWorkflowState(record); got != WorkflowStateDraft {
		t.Fatalf("GenreWorkflowState = %q, want %q", got, WorkflowStateDraft)
	}
}

func TestSetGenreWorkflowState_createsMinimalRecord(t *testing.T) {
	store := NewStore()
	record := SetGenreWorkflowState(store, "indie-rock", "Indie Rock", WorkflowStatePublishable)
	if record.Slug != "indie-rock" {
		t.Fatalf("Slug = %q", record.Slug)
	}
	if record.DisplayName != "Indie Rock" {
		t.Fatalf("DisplayName = %q", record.DisplayName)
	}
	if record.WorkflowState != WorkflowStatePublishable {
		t.Fatalf("WorkflowState = %q", record.WorkflowState)
	}
}

func TestUpsertGenreEditorial_preservesPublishableStateOnEnrichment(t *testing.T) {
	store := NewStore()
	// Promote genre to publishable.
	SetGenreWorkflowState(store, "hip-hop", "Hip-Hop", WorkflowStatePublishable)
	// Simulate enrichment: record has no workflow_state set (as NormalizeMatchedGenre produces).
	enrichRecord := GenreRecord{
		Slug:           "hip-hop",
		WikipediaTitle: "Hip-hop",
		WikipediaURL:   "https://en.wikipedia.org/wiki/Hip-hop",
		Summary:        "Hip-hop is a genre.",
		Status:         "matched",
	}
	result := UpsertGenreEditorial(store, enrichRecord)
	if result.WorkflowState != WorkflowStatePublishable {
		t.Fatalf("UpsertGenreEditorial reset workflow_state to %q; want %q", result.WorkflowState, WorkflowStatePublishable)
	}
}

func TestUpdate(t *testing.T) {
	store := NewStore()
	applyTestTaxonomy(store)
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

func TestUpdate_mergesSourceGenresAcrossUpdates(t *testing.T) {
	store := NewStore()
	applyTestTaxonomy(store)

	Update(store, "id1", "Artist One", "", []string{"rock"}, nil)
	record := Update(store, "id1", "Artist One", "", []string{"indie rock"}, nil)

	if len(record.SourceGenres) != 2 {
		t.Fatalf("SourceGenres = %v", record.SourceGenres)
	}
	if len(record.Genres) != 2 || record.Genres[0] != "indie-rock" || record.Genres[1] != "rock" {
		t.Fatalf("Genres = %v", record.Genres)
	}
}

func TestCanonicalArtistSlug_resolvesLegacyAlias(t *testing.T) {
	store := NewStore()
	Update(store, "id1", "R.E.M.", "", nil, nil)
	SetArtistSlugAlias(store, "rem", "r-e-m")

	if got := CanonicalArtistSlug(store, "rem"); got != "r-e-m" {
		t.Fatalf("CanonicalArtistSlug(rem) = %q", got)
	}
	if got := CanonicalArtistSlug(store, "r-e-m"); got != "r-e-m" {
		t.Fatalf("CanonicalArtistSlug(r-e-m) = %q", got)
	}
}

func TestResolvePlay_addsCanonicalIDs(t *testing.T) {
	store := NewStore()
	applyTestTaxonomy(store)
	play := ResolvePlay(store, models.Play{
		Source:           "spotify",
		TrackID:          "track1",
		TrackName:        "Song",
		ArtistID:         "artist1",
		ArtistName:       "Artist One",
		ArtistSpotifyURL: "https://open.spotify.com/artist/artist1",
		AdditionalArtists: []models.PlayArtist{
			{ID: "artist2", Name: "Artist Two", SpotifyURL: "https://open.spotify.com/artist/artist2"},
		},
		AlbumID:   "album1",
		AlbumName: "Album One",
	})

	if play.ArtistSlug != "artist-one" {
		t.Fatalf("ArtistSlug = %q", play.ArtistSlug)
	}
	if play.ReleaseSlug != "artist-one--album-one" {
		t.Fatalf("ReleaseSlug = %q", play.ReleaseSlug)
	}
	if play.TrackSlug != "artist-one--song" {
		t.Fatalf("TrackSlug = %q", play.TrackSlug)
	}
	if _, ok := store.Artists["artist-two"]; !ok {
		t.Fatalf("expected additional artist to be ensured, store=%v", store.Artists)
	}
}

func TestCanonicalizeTopArtist_tracksPendingGenres(t *testing.T) {
	store := NewStore()
	applyTestTaxonomy(store)
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
	applyTestTaxonomy(store)
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
	applyTestTaxonomy(store)
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
	applyTestTaxonomy(store)
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
	applyTestTaxonomy(store)
	Update(store, "a1", "Cached", "", []string{"rock"}, nil)
	plays := []models.Play{
		{
			ArtistID:   "a1",
			ArtistName: "Cached",
			AdditionalArtists: []models.PlayArtist{
				{ID: "a3", Name: "New2"},
			},
		},
		{ArtistID: "a2", ArtistName: "New1"},
		{ArtistID: "a2", ArtistName: "New1"},
		{ArtistID: "", ArtistName: "Empty", AdditionalArtists: []models.PlayArtist{{ID: "a5", Name: "New4"}}},
	}

	ids := UncachedArtistIDs(store, plays)
	if len(ids) != 3 {
		t.Fatalf("expected 3 uncached IDs, got %d: %v", len(ids), ids)
	}
	want := []string{"a2", "a3", "a5"}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("ids = %v, want %v", ids, want)
		}
	}
}
