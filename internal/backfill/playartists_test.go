package backfill

import (
	"testing"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
)

func TestCandidateTrackIDs_filtersMissingArtistArrays(t *testing.T) {
	plays := []models.Play{
		{PlayedAt: "2026-01-01T00:00:00Z", TrackID: "t1"},
		{PlayedAt: "2026-01-02T00:00:00Z", TrackID: "t2", AdditionalArtists: []models.PlayArtist{{ID: "a2"}}, AlbumID: "alb2"},
		{PlayedAt: "2025-01-02T00:00:00Z", TrackID: "t3"},
		{PlayedAt: "2026-01-03T00:00:00Z", TrackID: "t1"},
	}
	ids := CandidateTrackIDs(plays, PlayArtistsOptions{FromYear: "2026"})
	if len(ids) != 2 || ids[0] != "t1" || ids[1] != "t2" {
		t.Fatalf("ids = %v, want [t1 t2]", ids)
	}
}

func TestRewritePlayArtists_appliesTrackDetails(t *testing.T) {
	store := genres.NewStore()
	existing := []models.Play{{
		PlayedAt:   "2026-01-01T00:00:00Z",
		TrackID:    "t1",
		TrackName:  "Song",
		ArtistID:   "a1",
		ArtistName: "Artist One",
	}}
	tracks := []models.TrackDetails{{
		ID:        "t1",
		Name:      "Song",
		AlbumName: "Album One",
		Artists: []models.PlayArtist{
			{ID: "a1", Name: "Artist One"},
			{ID: "a2", Name: "Artist Two"},
		},
		AlbumArtists:     []models.PlayArtist{{ID: "a3", Name: "Artist Three"}},
		AlbumID:          "alb1",
		DiscNumber:       1,
		TrackNumber:      2,
		AlbumTotalTracks: 10,
	}}

	rewritten, changed := RewritePlayArtists(store, existing, tracks, PlayArtistsOptions{})
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	if len(rewritten[0].AdditionalArtists) != 1 || rewritten[0].AdditionalArtists[0].ID != "a2" {
		t.Fatalf("AdditionalArtists = %+v", rewritten[0].AdditionalArtists)
	}
	if rewritten[0].AlbumID != "alb1" {
		t.Fatalf("AlbumID = %q", rewritten[0].AlbumID)
	}
	if rewritten[0].TrackNumber != 2 || rewritten[0].AlbumTotalTracks != 10 {
		t.Fatalf("track position = %d/%d", rewritten[0].TrackNumber, rewritten[0].AlbumTotalTracks)
	}
}
