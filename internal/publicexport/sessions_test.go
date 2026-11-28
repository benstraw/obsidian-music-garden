package publicexport

import (
	"fmt"
	"testing"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
)

func TestDetectAlbumSessions_ToleratesSkipsAndTwoInterruptions(t *testing.T) {
	catalog := sessionCatalog(8)
	base := time.Date(2026, 8, 1, 23, 50, 0, 0, time.UTC)
	plays := []models.Play{
		sessionPlay(base, 1),
		sessionPlay(base.Add(4*time.Minute), 2),
		{PlayedAt: base.Add(8 * time.Minute).Format(time.RFC3339), Source: "spotify", TrackName: "Interrupt"},
		{PlayedAt: base.Add(12 * time.Minute).Format(time.RFC3339), Source: "spotify", TrackName: "Interrupt 2"},
		sessionPlay(base.Add(16*time.Minute), 4),
		sessionPlay(base.Add(20*time.Minute), 6),
	}
	sessions := DetectAlbumSessions(plays, catalog, time.UTC)
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v", sessions)
	}
	if sessions[0].ObservedTracks != 4 || sessions[0].CoveragePercent != 50 {
		t.Fatalf("session = %+v", sessions[0])
	}
}

func TestDetectAlbumSessions_RejectsGapBackwardAndLegacy(t *testing.T) {
	catalog := sessionCatalog(6)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for name, plays := range map[string][]models.Play{
		"gap":      {sessionPlay(base, 1), sessionPlay(base.Add(31*time.Minute), 2), sessionPlay(base.Add(35*time.Minute), 3)},
		"backward": {sessionPlay(base, 3), sessionPlay(base.Add(time.Minute), 2), sessionPlay(base.Add(2*time.Minute), 4)},
		"legacy": func() []models.Play {
			p := []models.Play{sessionPlay(base, 1), sessionPlay(base.Add(time.Minute), 2), sessionPlay(base.Add(2*time.Minute), 3)}
			for i := range p {
				p[i].Source = "legacy-backfill"
			}
			return p
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			if got := DetectAlbumSessions(plays, catalog, time.UTC); len(got) != 0 {
				t.Fatalf("sessions = %+v", got)
			}
		})
	}
}

func sessionCatalog(total int) *genres.Catalog {
	catalog := genres.NewCatalog()
	catalog.Releases.Releases["artist--album"] = genres.ReleaseRecord{
		Slug: "artist--album", Name: "Album",
		Editions: []genres.ReleaseEdition{{SpotifyAlbumID: "album-1", Name: "Album", TotalTracks: total}},
	}
	return catalog
}

func sessionPlay(at time.Time, track int) models.Play {
	return models.Play{
		PlayedAt: at.Format(time.RFC3339), Source: "spotify", ReleaseSlug: "artist--album",
		AlbumID: "album-1", TrackID: fmt.Sprintf("track-%d", track), TrackName: fmt.Sprintf("Track %d", track),
		DiscNumber: 1, TrackNumber: track,
	}
}
