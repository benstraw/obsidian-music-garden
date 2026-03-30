package importlegacyplays

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
)

type Options struct {
	SourceDir    string
	ManifestPath string
	DryRun       bool
	Force        bool
	ArtistFilter string
	FallbackFrom time.Time
	FallbackTo   time.Time
}

type Summary struct {
	SourceItems      int
	PreparedPlays    int
	ArtistsWithDates int
	ArtistsFallback  int
	WeeksTouched     int
}

type Manifest struct {
	SourceDir       string    `json:"source_dir"`
	TopTracksPath   string    `json:"top_tracks_path"`
	TopTracksSHA256 string    `json:"top_tracks_sha256"`
	ArtistsPath     string    `json:"artists_path,omitempty"`
	ArtistsSHA256   string    `json:"artists_sha256,omitempty"`
	ImportedAt      time.Time `json:"imported_at"`
	PreparedPlays   int       `json:"prepared_plays"`
	DateMode        string    `json:"date_mode"`
	FallbackFrom    string    `json:"fallback_from,omitempty"`
	FallbackTo      string    `json:"fallback_to,omitempty"`
	ArtistFilter    string    `json:"artist_filter,omitempty"`
}

type Result struct {
	Plays    []models.Play
	Summary  Summary
	Manifest Manifest
}

type topTracksFile struct {
	Items []legacyTrack `json:"items"`
}

type legacyTrack struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Artists      []legacyTrackArtist `json:"artists"`
	Album        legacyAlbum         `json:"album"`
	DurationMS   int                 `json:"duration_ms"`
	ExternalURLs map[string]string   `json:"external_urls"`
}

type legacyTrackArtist struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type legacyAlbum struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	Artists []legacyTrackArtist `json:"artists"`
}

type legacyArtistsFileRecord struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	SpotifyURL string   `json:"spotify_url"`
	Genres     []string `json:"genres"`
	FirstSeen  string   `json:"first_seen"`
	LastSeen   string   `json:"last_seen"`
}

func Prepare(catalog *genres.Catalog, opts Options) (Result, error) {
	sourceDir := strings.TrimSpace(opts.SourceDir)
	if sourceDir == "" {
		return Result{}, fmt.Errorf("empty source dir")
	}
	if opts.FallbackFrom.IsZero() || opts.FallbackTo.IsZero() {
		return Result{}, fmt.Errorf("fallback date window is required")
	}
	if opts.FallbackTo.Before(opts.FallbackFrom) {
		return Result{}, fmt.Errorf("fallback-to before fallback-from")
	}
	if err := checkManifest(opts.ManifestPath, opts.Force); err != nil {
		return Result{}, err
	}

	topTracksPath := filepath.Join(sourceDir, "topTracks.json")
	topTracksBytes, err := os.ReadFile(topTracksPath)
	if err != nil {
		return Result{}, fmt.Errorf("read topTracks.json: %w", err)
	}
	var topTracks topTracksFile
	if err := json.Unmarshal(topTracksBytes, &topTracks); err != nil {
		return Result{}, fmt.Errorf("decode topTracks.json: %w", err)
	}

	artistsPath := filepath.Join(sourceDir, "artists.json")
	artistHints, artistsHash, err := loadArtistHints(artistsPath)
	if err != nil {
		return Result{}, err
	}

	filter := strings.TrimSpace(opts.ArtistFilter)
	items := filterTracks(topTracks.Items, filter)
	grouped := groupByPrimaryArtist(items)
	artistIDs := make([]string, 0, len(grouped))
	for artistID := range grouped {
		artistIDs = append(artistIDs, artistID)
	}
	sort.Strings(artistIDs)

	playsOut := make([]models.Play, 0, len(items))
	artistsWithDates := 0
	artistsFallback := 0
	weeksTouched := map[string]bool{}

	for _, artistID := range artistIDs {
		group := grouped[artistID]
		windowFrom, windowTo, usedHint := artistWindow(group[0], artistHints, opts.FallbackFrom, opts.FallbackTo)
		if usedHint {
			artistsWithDates++
		} else {
			artistsFallback++
		}
		for i, track := range group {
			playedAt := distributedPlayedAt(windowFrom, windowTo, track, i, len(group))
			play := playFromLegacyTrack(track, playedAt)
			play = genres.ResolvePlay(catalog, play)
			playsOut = append(playsOut, play)
			if t, err := time.Parse(time.RFC3339Nano, playedAt); err == nil {
				weeksTouched[isoWeekKey(t)] = true
			}
		}
	}

	manifest := Manifest{
		SourceDir:       sourceDir,
		TopTracksPath:   topTracksPath,
		TopTracksSHA256: sha256Hex(topTracksBytes),
		ArtistsPath:     artistsPath,
		ArtistsSHA256:   artistsHash,
		PreparedPlays:   len(playsOut),
		DateMode:        "approx-real",
		FallbackFrom:    opts.FallbackFrom.Format("2006-01-02"),
		FallbackTo:      opts.FallbackTo.Format("2006-01-02"),
		ArtistFilter:    filter,
	}

	return Result{
		Plays: playsOut,
		Summary: Summary{
			SourceItems:      len(items),
			PreparedPlays:    len(playsOut),
			ArtistsWithDates: artistsWithDates,
			ArtistsFallback:  artistsFallback,
			WeeksTouched:     len(weeksTouched),
		},
		Manifest: manifest,
	}, nil
}

func WriteManifest(path string, manifest Manifest) error {
	manifest.ImportedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func checkManifest(path string, force bool) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if force {
		return nil
	}
	return fmt.Errorf("legacy play import manifest already exists at %s (use --force to override)", path)
}

func loadArtistHints(path string) (map[string]legacyArtistsFileRecord, string, error) {
	result := map[string]legacyArtistsFileRecord{}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return result, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read artists.json: %w", err)
	}
	var payload map[string]legacyArtistsFileRecord
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, "", fmt.Errorf("decode artists.json: %w", err)
	}
	for _, record := range payload {
		if strings.TrimSpace(record.ID) != "" {
			result[record.ID] = record
		}
	}
	return result, sha256Hex(data), nil
}

func filterTracks(items []legacyTrack, filter string) []legacyTrack {
	if filter == "" {
		return append([]legacyTrack(nil), items...)
	}
	filter = strings.ToLower(strings.TrimSpace(filter))
	out := make([]legacyTrack, 0, len(items))
	for _, item := range items {
		if len(item.Artists) == 0 {
			continue
		}
		primary := item.Artists[0]
		if strings.EqualFold(primary.ID, filter) || strings.EqualFold(primary.Name, filter) || genres.Slug(primary.Name) == genres.Slug(filter) {
			out = append(out, item)
		}
	}
	return out
}

func groupByPrimaryArtist(items []legacyTrack) map[string][]legacyTrack {
	grouped := map[string][]legacyTrack{}
	for _, item := range items {
		if len(item.Artists) == 0 {
			continue
		}
		primaryID := strings.TrimSpace(item.Artists[0].ID)
		if primaryID == "" {
			primaryID = "name:" + strings.TrimSpace(item.Artists[0].Name)
		}
		grouped[primaryID] = append(grouped[primaryID], item)
	}
	return grouped
}

func artistWindow(track legacyTrack, hints map[string]legacyArtistsFileRecord, fallbackFrom, fallbackTo time.Time) (time.Time, time.Time, bool) {
	fallbackStart, fallbackEnd := normalizeWindow(fallbackFrom, fallbackTo)
	if len(track.Artists) == 0 {
		return fallbackStart, fallbackEnd, false
	}
	record, ok := hints[strings.TrimSpace(track.Artists[0].ID)]
	if !ok {
		return fallbackStart, fallbackEnd, false
	}
	first, firstOK := parseLegacyDate(record.FirstSeen)
	last, lastOK := parseLegacyDate(record.LastSeen)
	clamp := func(from, to time.Time) (time.Time, time.Time, bool) {
		start, end := normalizeWindow(from, to)
		if end.Before(fallbackStart) || start.After(fallbackEnd) {
			return fallbackStart, fallbackEnd, false
		}
		if start.Before(fallbackStart) {
			start = fallbackStart
		}
		if end.After(fallbackEnd) {
			end = fallbackEnd
		}
		return start, end, true
	}
	switch {
	case firstOK && lastOK:
		return clamp(first, last)
	case firstOK:
		return clamp(first.AddDate(0, 0, -30), first.AddDate(0, 0, 30))
	case lastOK:
		return clamp(last.AddDate(0, 0, -30), last.AddDate(0, 0, 30))
	default:
		return fallbackStart, fallbackEnd, false
	}
}

func normalizeWindow(from, to time.Time) (time.Time, time.Time) {
	start := time.Date(from.UTC().Year(), from.UTC().Month(), from.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.UTC().Year(), to.UTC().Month(), to.UTC().Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
	if end.Before(start) {
		return end, start
	}
	return trimISOYearWindow(start, end)
}

func trimISOYearWindow(start, end time.Time) (time.Time, time.Time) {
	targetStartYear := start.UTC().Year()
	for start.Before(end) {
		isoYear, _ := start.UTC().ISOWeek()
		if isoYear == targetStartYear {
			break
		}
		start = time.Date(start.UTC().Year(), start.UTC().Month(), start.UTC().Day()+1, 0, 0, 0, 0, time.UTC)
	}

	targetEndYear := end.UTC().Year()
	for end.After(start) {
		isoYear, _ := end.UTC().ISOWeek()
		if isoYear == targetEndYear {
			break
		}
		prev := end.AddDate(0, 0, -1)
		end = time.Date(prev.UTC().Year(), prev.UTC().Month(), prev.UTC().Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
	}

	if end.Before(start) {
		return start, start
	}
	return start, end
}

func distributedPlayedAt(from, to time.Time, track legacyTrack, index, total int) string {
	if total <= 0 {
		total = 1
	}
	window := to.Sub(from)
	step := time.Duration(0)
	if total > 1 && window > 0 {
		step = window / time.Duration(total)
	}
	t := from
	if step > 0 {
		t = from.Add(step * time.Duration(index))
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", track.ID, primaryArtistID(track), index)))
	ns := int64(h[0])<<24 | int64(h[1])<<16 | int64(h[2])<<8 | int64(h[3])
	ns = ns % int64(time.Second)
	t = t.Add(time.Duration(ns))
	return t.UTC().Format(time.RFC3339Nano)
}

func playFromLegacyTrack(track legacyTrack, playedAt string) models.Play {
	play := models.Play{
		PlayedAt:        playedAt,
		Source:          "legacy-backfill",
		TrackID:         strings.TrimSpace(track.ID),
		TrackName:       strings.TrimSpace(track.Name),
		AlbumID:         strings.TrimSpace(track.Album.ID),
		AlbumName:       strings.TrimSpace(track.Album.Name),
		DurationMS:      track.DurationMS,
		TrackSpotifyURL: strings.TrimSpace(track.ExternalURLs["spotify"]),
	}
	if len(track.Artists) > 0 {
		play.ArtistID = strings.TrimSpace(track.Artists[0].ID)
		play.ArtistName = strings.TrimSpace(track.Artists[0].Name)
		play.ArtistSpotifyURL = strings.TrimSpace(track.Artists[0].ExternalURLs["spotify"])
		if len(track.Artists) > 1 {
			play.AdditionalArtists = make([]models.PlayArtist, 0, len(track.Artists)-1)
			for _, artist := range track.Artists[1:] {
				play.AdditionalArtists = append(play.AdditionalArtists, models.PlayArtist{
					ID:         strings.TrimSpace(artist.ID),
					Name:       strings.TrimSpace(artist.Name),
					SpotifyURL: strings.TrimSpace(artist.ExternalURLs["spotify"]),
				})
			}
		}
	}
	return play
}

func primaryArtistID(track legacyTrack) string {
	if len(track.Artists) == 0 {
		return ""
	}
	return strings.TrimSpace(track.Artists[0].ID)
}

func parseLegacyDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func isoWeekKey(t time.Time) string {
	year, week := t.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
