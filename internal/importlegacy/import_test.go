package importlegacy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benstraw/music-garden/internal/genres"
)

func TestRun_importsLegacyTrackCounts(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "data", "spotify")
	if err := os.MkdirAll(filepath.Join(root, "public", "musical-genres", "ambient"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(sourceDir, "topArtists.json"), `{"items":[{"id":"a1","name":"Artist One","genres":["ambient"],"external_urls":{"spotify":"https://open.spotify.com/artist/a1"},"images":[]}]}`)
	mustWrite(t, filepath.Join(sourceDir, "snapshot-2024-06.json"), `{"items":[]}`)
	mustWrite(t, filepath.Join(sourceDir, "artists.json"), `{"artist-one":{"id":"a1","name":"Artist One","spotify_url":"https://open.spotify.com/artist/a1","genres":["ambient"]}}`)
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[
		{"id":"t1","name":"Song One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}},{"id":"a2","name":"Artist Two","external_urls":{"spotify":"https://open.spotify.com/artist/a2"}}],"album":{"id":"alb1","name":"Album One","artists":[{"id":"a3","name":"Artist Three","external_urls":{"spotify":"https://open.spotify.com/artist/a3"}}]},"external_urls":{"spotify":"https://open.spotify.com/track/t1"},"duration_ms":1000},
		{"id":"t1","name":"Song One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}},{"id":"a2","name":"Artist Two","external_urls":{"spotify":"https://open.spotify.com/artist/a2"}}],"album":{"id":"alb1","name":"Album One","artists":[{"id":"a3","name":"Artist Three","external_urls":{"spotify":"https://open.spotify.com/artist/a3"}}]},"external_urls":{"spotify":"https://open.spotify.com/track/t1"},"duration_ms":1000}
	]}`)

	store := genres.NewStore()
	store.GenreAliases["ambient"] = "ambient"
	summary, err := Run(store, Options{
		SourceDir: sourceDir,
		DataRoot:  filepath.Join(root, "garden-data"),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.TracksAdded != 1 {
		t.Fatalf("TracksAdded = %d, want 1", summary.TracksAdded)
	}
	track := store.Tracks["artist-one--song-one"]
	if track.LegacyPlayCount != 2 {
		t.Fatalf("LegacyPlayCount = %d, want 2", track.LegacyPlayCount)
	}
	if len(track.AdditionalArtistSlugs) != 0 {
		t.Fatalf("AdditionalArtistSlugs = %v, want none for unknown collaborators", track.AdditionalArtistSlugs)
	}
	if len(track.AlbumArtistSlugs) != 0 {
		t.Fatalf("AlbumArtistSlugs = %v, want none for unknown collaborators", track.AlbumArtistSlugs)
	}
	if _, ok := store.Artists["artist-two"]; ok {
		t.Fatalf("unexpected collaborator artist-two in store")
	}
	if _, ok := store.Artists["artist-three"]; ok {
		t.Fatalf("unexpected album collaborator artist-three in store")
	}
}

func TestRun_importsAllowedCollaboratorsFromLegacyTracks(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "data", "spotify")
	if err := os.MkdirAll(filepath.Join(root, "public", "musical-genres", "ambient"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(sourceDir, "topArtists.json"), `{"items":[
		{"id":"a1","name":"Artist One","genres":["ambient"],"external_urls":{"spotify":"https://open.spotify.com/artist/a1"},"images":[]},
		{"id":"a2","name":"Artist Two","genres":["ambient"],"external_urls":{"spotify":"https://open.spotify.com/artist/a2"},"images":[]}
	]}`)
	mustWrite(t, filepath.Join(sourceDir, "snapshot-2024-06.json"), `{"items":[]}`)
	mustWrite(t, filepath.Join(sourceDir, "artists.json"), `{"artist-one":{"id":"a1","name":"Artist One","spotify_url":"https://open.spotify.com/artist/a1","genres":["ambient"]},"artist-three":{"id":"a3","name":"Artist Three","spotify_url":"https://open.spotify.com/artist/a3","genres":["ambient"]}}`)
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[
		{"id":"t1","name":"Song One","artists":[{"id":"a1","name":"Artist One","external_urls":{"spotify":"https://open.spotify.com/artist/a1"}},{"id":"a2","name":"Artist Two","external_urls":{"spotify":"https://open.spotify.com/artist/a2"}}],"album":{"id":"alb1","name":"Album One","artists":[{"id":"a3","name":"Artist Three","external_urls":{"spotify":"https://open.spotify.com/artist/a3"}}]},"external_urls":{"spotify":"https://open.spotify.com/track/t1"},"duration_ms":1000}
	]}`)

	store := genres.NewStore()
	store.GenreAliases["ambient"] = "ambient"
	_, err := Run(store, Options{
		SourceDir: sourceDir,
		DataRoot:  filepath.Join(root, "garden-data"),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	track := store.Tracks["artist-one--song-one"]
	if len(track.AdditionalArtistSlugs) != 1 || track.AdditionalArtistSlugs[0] != "artist-two" {
		t.Fatalf("AdditionalArtistSlugs = %v", track.AdditionalArtistSlugs)
	}
	if len(track.AlbumArtistSlugs) != 1 || track.AlbumArtistSlugs[0] != "artist-three" {
		t.Fatalf("AlbumArtistSlugs = %v", track.AlbumArtistSlugs)
	}
}

func TestRun_recordsLegacyArtistSlugAliases(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "data", "spotify")
	if err := os.MkdirAll(filepath.Join(root, "public", "musical-genres", "ambient"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(sourceDir, "topArtists.json"), `{"items":[]}`)
	mustWrite(t, filepath.Join(sourceDir, "snapshot-2024-06.json"), `{"items":[]}`)
	mustWrite(t, filepath.Join(sourceDir, "artists.json"), `{"rem":{"id":"a1","name":"R.E.M.","spotify_url":"https://open.spotify.com/artist/a1","genres":["ambient"]}}`)
	mustWrite(t, filepath.Join(sourceDir, "topTracks.json"), `{"items":[]}`)

	store := genres.NewStore()
	store.GenreAliases["ambient"] = "ambient"
	genres.UpsertArtistMetadata(store, "a1", "R.E.M.", "https://open.spotify.com/artist/a1", "", []string{"ambient"}, nil)

	if _, err := Run(store, Options{SourceDir: sourceDir, DataRoot: filepath.Join(root, "garden-data")}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := genres.CanonicalArtistSlug(store, "rem"); got != "r-e-m" {
		t.Fatalf("CanonicalArtistSlug(rem) = %q", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
