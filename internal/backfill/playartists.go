package backfill

import (
	"reflect"
	"sort"
	"strings"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
	"github.com/benstraw/music-garden/internal/plays"
)

type PlayArtistsOptions struct {
	FromYear string
	Limit    int
	DryRun   bool
	Verbose  bool
}

type PlayArtistsSummary struct {
	CandidateTrackIDs int
	UpdatedPlays      int
}

// RewritePlayArtists rewrites plays in-memory using fetched track metadata and
// returns the updated plays plus the count of changed records.
func RewritePlayArtists(source any, existing []models.Play, tracks []models.TrackDetails, opts PlayArtistsOptions) ([]models.Play, int) {
	trackByID := make(map[string]models.TrackDetails, len(tracks))
	for _, track := range tracks {
		if strings.TrimSpace(track.ID) == "" {
			continue
		}
		trackByID[track.ID] = track
	}

	updated := make([]models.Play, len(existing))
	changed := 0
	for i, play := range existing {
		next := play
		track, ok := trackByID[play.TrackID]
		if ok {
			next = applyTrackDetails(next, track)
		}
		next = genres.ResolvePlay(source, next)
		if !reflect.DeepEqual(next, play) {
			changed++
		}
		updated[i] = next
	}
	return updated, changed
}

func CandidateTrackIDs(existing []models.Play, opts PlayArtistsOptions) []string {
	seen := map[string]bool{}
	var ids []string
	for _, play := range existing {
		if opts.FromYear != "" && !strings.HasPrefix(play.PlayedAt, opts.FromYear+"-") {
			continue
		}
		if strings.TrimSpace(play.TrackID) == "" || seen[play.TrackID] {
			continue
		}
		if len(play.AdditionalArtists) > 0 && play.AlbumID != "" {
			continue
		}
		seen[play.TrackID] = true
		ids = append(ids, play.TrackID)
	}
	sort.Strings(ids)
	if opts.Limit > 0 && len(ids) > opts.Limit {
		return ids[:opts.Limit]
	}
	return ids
}

func applyTrackDetails(play models.Play, track models.TrackDetails) models.Play {
	if play.TrackID == "" || play.TrackID != track.ID {
		return play
	}
	if play.TrackName == "" {
		play.TrackName = track.Name
	}
	if play.TrackSpotifyURL == "" {
		play.TrackSpotifyURL = track.TrackSpotifyURL
	}
	if play.AlbumID == "" {
		play.AlbumID = track.AlbumID
	}
	if play.AlbumName == "" {
		play.AlbumName = track.AlbumName
	}
	if play.DurationMS == 0 {
		play.DurationMS = track.DurationMS
	}
	if len(track.Artists) > 0 {
		play.ArtistID = firstNonEmpty(play.ArtistID, track.Artists[0].ID)
		play.ArtistName = firstNonEmpty(play.ArtistName, track.Artists[0].Name)
		play.ArtistSpotifyURL = firstNonEmpty(play.ArtistSpotifyURL, track.Artists[0].SpotifyURL)
		if len(play.AdditionalArtists) == 0 && len(track.Artists) > 1 {
			play.AdditionalArtists = append([]models.PlayArtist(nil), track.Artists[1:]...)
		}
	}
	return play
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// SaveRewrittenPlays persists rewritten plays back into sharded files.
func SaveRewrittenPlays(playsDir string, rewritten []models.Play) error {
	_, err := plays.SaveSharded(playsDir, rewritten)
	return err
}
