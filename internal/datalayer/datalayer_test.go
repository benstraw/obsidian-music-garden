package datalayer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
)

func TestEnsureLayout(t *testing.T) {
	root := t.TempDir()
	if err := EnsureLayout(root); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}

	paths := []string{
		filepath.Join(root, "raw", "spotify", "recently-played"),
		filepath.Join(root, "raw", "wikipedia", "search"),
		filepath.Join(root, "normalized", "artists"),
		filepath.Join(root, "normalized", "tracks"),
		filepath.Join(root, "aggregated", "tracks"),
		filepath.Join(root, "aggregated", "genres"),
	}
	for _, path := range paths {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s to exist", path)
		}
	}
}

func TestWriteRawSpotifyRecentlyPlayed(t *testing.T) {
	root := t.TempDir()
	ts := time.Date(2026, 3, 27, 8, 9, 10, 0, time.UTC)
	path, err := WriteRawSpotifyRecentlyPlayed(root, ts, []byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("WriteRawSpotifyRecentlyPlayed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != `{"items":[]}` {
		t.Fatalf("raw body mismatch: %s", string(data))
	}
}

func TestSyncAggregatedStore(t *testing.T) {
	root := t.TempDir()
	store := genres.NewStore()
	genres.ApplyTaxonomy(store, &genres.Taxonomy{
		Version: 1,
		Genres: []genres.GenreDefinition{
			{Slug: "indie-rock", DisplayName: "Indie Rock", Aliases: []string{"indie rock"}},
		},
	})
	artist := genres.Update(store, "artist1", "Artist One", "https://open.spotify.com/artist/artist1", []string{"indie rock"}, nil)
	artist.MusicBrainzArtistID = "mb-artist-1"
	store.Artists[artist.Slug] = artist
	genres.UpsertGenreEditorial(store, genres.GenreRecord{
		Slug:           "indie-rock",
		DisplayName:    "Indie Rock",
		WorkflowState:  genres.WorkflowStatePublishable,
		WikipediaTitle: "Indie rock",
		WikipediaURL:   "https://en.wikipedia.org/wiki/Indie_rock",
		Summary:        "Indie rock is a rock music genre.",
		Status:         "matched",
		Attribution: &genres.GenreSourceAttribution{
			Source:      "wikipedia",
			PageTitle:   "Indie rock",
			PageURL:     "https://en.wikipedia.org/wiki/Indie_rock",
			RetrievedAt: "2026-03-27",
		},
	})
	play := genres.ResolvePlay(store, models.Play{
		PlayedAt:   "2026-03-27T12:00:00Z",
		TrackID:    "track1",
		ArtistName: "Artist One",
		ArtistID:   "artist1",
		TrackName:  "Song One",
		AlbumName:  "Album One",
		AlbumID:    "album1",
	})
	plays := []models.Play{play}

	if err := SyncAggregatedStore(root, store, plays); err != nil {
		t.Fatalf("SyncAggregatedStore: %v", err)
	}

	checks := []string{
		filepath.Join(root, "normalized", "artists", "artist-one.json"),
		filepath.Join(root, "normalized", "tracks", "artist-one--song-one.json"),
		filepath.Join(root, "aggregated", "artists", "artist-one.json"),
		filepath.Join(root, "aggregated", "releases", "artist-one--album-one.json"),
		filepath.Join(root, "aggregated", "tracks", "artist-one--song-one.json"),
		filepath.Join(root, "aggregated", "genres", "indie-rock.json"),
	}
	for _, path := range checks {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected aggregated file %s: %v", path, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(root, "aggregated", "genres", "indie-rock.json"))
	if err != nil {
		t.Fatalf("ReadFile aggregated genre: %v", err)
	}
	var record AggregatedGenreRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("Unmarshal aggregated genre: %v", err)
	}
	if record.DisplayTitle != "Indie Rock" {
		t.Fatalf("DisplayTitle = %q", record.DisplayTitle)
	}
	if record.WorkflowState != genres.WorkflowStatePublishable {
		t.Fatalf("WorkflowState = %q", record.WorkflowState)
	}
	if record.ListeningStats.PlayCount != 1 {
		t.Fatalf("PlayCount = %d", record.ListeningStats.PlayCount)
	}
	if len(record.TopArtists) != 1 || record.TopArtists[0].CanonicalSlug != "artist-one" || record.TopArtists[0].PlayCount != 1 {
		t.Fatalf("TopArtists = %+v", record.TopArtists)
	}
	if len(record.TopReleases) != 1 || record.TopReleases[0].CanonicalSlug != "artist-one--album-one" || record.TopReleases[0].PlayCount != 1 {
		t.Fatalf("TopReleases = %+v", record.TopReleases)
	}
	if len(record.TopTracks) != 1 || record.TopTracks[0].CanonicalSlug != "artist-one--song-one" || record.TopTracks[0].PlayCount != 1 {
		t.Fatalf("TopTracks = %+v", record.TopTracks)
	}
	if len(record.SourceRefs) < 3 {
		t.Fatalf("SourceRefs = %+v", record.SourceRefs)
	}

	artistData, err := os.ReadFile(filepath.Join(root, "aggregated", "artists", "artist-one.json"))
	if err != nil {
		t.Fatalf("ReadFile aggregated artist: %v", err)
	}
	var artistRecord AggregatedArtistRecord
	if err := json.Unmarshal(artistData, &artistRecord); err != nil {
		t.Fatalf("Unmarshal aggregated artist: %v", err)
	}
	if artistRecord.PlayCount != 1 || artistRecord.LegacyPlayCount != 0 {
		t.Fatalf("AggregatedArtist counts = %+v", artistRecord)
	}

	releaseData, err := os.ReadFile(filepath.Join(root, "aggregated", "releases", "artist-one--album-one.json"))
	if err != nil {
		t.Fatalf("ReadFile aggregated release: %v", err)
	}
	var releaseRecord AggregatedReleaseRecord
	if err := json.Unmarshal(releaseData, &releaseRecord); err != nil {
		t.Fatalf("Unmarshal aggregated release: %v", err)
	}
	if releaseRecord.PlayCount != 1 || releaseRecord.LegacyPlayCount != 0 {
		t.Fatalf("AggregatedRelease counts = %+v", releaseRecord)
	}

	trackData, err := os.ReadFile(filepath.Join(root, "aggregated", "tracks", "artist-one--song-one.json"))
	if err != nil {
		t.Fatalf("ReadFile aggregated track: %v", err)
	}
	var trackRecord AggregatedTrackRecord
	if err := json.Unmarshal(trackData, &trackRecord); err != nil {
		t.Fatalf("Unmarshal aggregated track: %v", err)
	}
	if trackRecord.PlayCount != 1 || trackRecord.LegacyPlayCount != 0 {
		t.Fatalf("AggregatedTrack counts = %+v", trackRecord)
	}
}

func TestRebuildAggregatedGenre_canonicalMappedGenreIsNotPending(t *testing.T) {
	root := t.TempDir()
	store := genres.NewStore()
	store.GenreAliases["classical"] = "classical"
	store.PendingGenreAliases = []string{"classical"}

	path, err := RebuildAggregatedGenre(root, store, nil, "classical")
	if err != nil {
		t.Fatalf("RebuildAggregatedGenre: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var record AggregatedGenreRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if record.Pending {
		t.Fatalf("Pending = %v, want false", record.Pending)
	}
}

func TestSyncAggregatedStore_canonicalMappedGenreRemainsNotPending(t *testing.T) {
	root := t.TempDir()
	store := genres.NewStore()
	store.GenreAliases["jazz"] = "jazz"
	store.PendingGenreAliases = []string{"jazz"}
	genres.UpsertGenreEditorial(store, genres.GenreRecord{
		Slug:           "jazz",
		DisplayName:    "Jazz",
		WorkflowState:  genres.WorkflowStatePublishable,
		WikipediaTitle: "Jazz",
		WikipediaURL:   "https://en.wikipedia.org/wiki/Jazz",
		Summary:        "Jazz is a music genre.",
		Status:         "matched",
	})

	if err := SyncAggregatedStore(root, store, nil); err != nil {
		t.Fatalf("SyncAggregatedStore: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "aggregated", "genres", "jazz.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var record AggregatedGenreRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if record.Pending {
		t.Fatalf("Pending = %v, want false", record.Pending)
	}
	if record.WorkflowState != genres.WorkflowStatePublishable {
		t.Fatalf("WorkflowState = %q, want %q", record.WorkflowState, genres.WorkflowStatePublishable)
	}
}
