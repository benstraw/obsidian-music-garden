package publicexport

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
)

type AlbumSession struct {
	ReleaseSlug     string `json:"release_slug"`
	EditionID       string `json:"edition_id,omitempty"`
	Date            string `json:"date"`
	ObservedTracks  int    `json:"observed_tracks"`
	TotalTracks     int    `json:"total_tracks"`
	CoveragePercent int    `json:"coverage_percent"`
}

type timedPlay struct {
	play models.Play
	time time.Time
}

func DetectAlbumSessions(input []models.Play, catalog *genres.Catalog, loc *time.Location) []AlbumSession {
	plays := make([]timedPlay, 0, len(input))
	for _, play := range input {
		if !isRealPlay(play) {
			continue
		}
		playedAt, err := time.Parse(time.RFC3339Nano, play.PlayedAt)
		if err != nil {
			continue
		}
		plays = append(plays, timedPlay{play: play, time: playedAt})
	}
	sort.Slice(plays, func(i, j int) bool { return plays[i].time.Before(plays[j].time) })

	var sessions []AlbumSession
	for start := 0; start < len(plays); {
		release, edition, ok := releaseEditionForPlay(catalog, plays[start].play)
		if !ok || edition.TotalTracks < 3 {
			start++
			continue
		}
		firstPosition := trackPosition(plays[start].play)
		if firstPosition <= 0 {
			start++
			continue
		}

		positions := map[int]bool{firstPosition: true}
		lastPosition := firstPosition
		lastEvent := plays[start].time
		unrelated := 0
		end := start
		for i := start + 1; i < len(plays); i++ {
			if plays[i].time.Sub(lastEvent) > 30*time.Minute {
				break
			}
			lastEvent = plays[i].time
			otherRelease, otherEdition, sameRelease := releaseEditionForPlay(catalog, plays[i].play)
			if !sameRelease || otherRelease.Slug != release.Slug || otherEdition.SpotifyAlbumID != edition.SpotifyAlbumID {
				unrelated++
				if unrelated > 2 {
					break
				}
				end = i
				continue
			}
			unrelated = 0
			position := trackPosition(plays[i].play)
			if position <= 0 {
				continue
			}
			if positions[position] {
				end = i
				continue
			}
			if position < lastPosition {
				break
			}
			positions[position] = true
			lastPosition = position
			end = i
		}

		threshold := int(math.Ceil(float64(edition.TotalTracks) * 0.5))
		if threshold < 3 {
			threshold = 3
		}
		if len(positions) >= threshold {
			date := plays[start].time.In(loc).Format("2006-01-02")
			sessions = append(sessions, AlbumSession{
				ReleaseSlug:     release.Slug,
				EditionID:       edition.SpotifyAlbumID,
				Date:            date,
				ObservedTracks:  len(positions),
				TotalTracks:     edition.TotalTracks,
				CoveragePercent: int(math.Round(float64(len(positions)) * 100 / float64(edition.TotalTracks))),
			})
			start = end + 1
			continue
		}
		start++
	}
	return sessions
}

func releaseEditionForPlay(catalog *genres.Catalog, play models.Play) (genres.ReleaseRecord, genres.ReleaseEdition, bool) {
	if strings.TrimSpace(play.ReleaseSlug) == "" {
		return genres.ReleaseRecord{}, genres.ReleaseEdition{}, false
	}
	release, ok := catalog.Releases.Releases[play.ReleaseSlug]
	if !ok {
		return genres.ReleaseRecord{}, genres.ReleaseEdition{}, false
	}
	for _, edition := range release.Editions {
		if edition.SpotifyAlbumID == play.AlbumID {
			return release, edition, true
		}
	}
	if len(release.Editions) == 1 && play.AlbumID == "" {
		return release, release.Editions[0], true
	}
	return genres.ReleaseRecord{}, genres.ReleaseEdition{}, false
}

func trackPosition(play models.Play) int {
	if play.TrackNumber <= 0 {
		return 0
	}
	disc := play.DiscNumber
	if disc <= 0 {
		disc = 1
	}
	return disc*1000 + play.TrackNumber
}

func isRealPlay(play models.Play) bool {
	source := strings.ToLower(strings.TrimSpace(play.Source))
	return !strings.Contains(source, "legacy") && !strings.Contains(source, "synthetic")
}
