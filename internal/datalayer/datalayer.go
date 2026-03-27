package datalayer

import (
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

// NormalizedArtistRecord models a source-cleaned artist before cross-source merge.
type NormalizedArtistRecord struct {
	Source              string   `json:"source"`
	SourceArtistID      string   `json:"source_artist_id,omitempty"`
	Name                string   `json:"name"`
	SpotifyURL          string   `json:"spotify_url,omitempty"`
	MusicBrainzArtistID string   `json:"musicbrainz_artist_id,omitempty"`
	SourceGenres        []string `json:"source_genres,omitempty"`
	CanonicalGenreSlugs []string `json:"canonical_genre_slugs,omitempty"`
}

// NormalizedReleaseRecord models a source-cleaned release before cross-source merge.
type NormalizedReleaseRecord struct {
	Source                     string `json:"source"`
	SourceReleaseID            string `json:"source_release_id,omitempty"`
	Name                       string `json:"name"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	MusicBrainzReleaseID       string `json:"musicbrainz_release_id,omitempty"`
	MusicBrainzReleaseGroupID  string `json:"musicbrainz_release_group_id,omitempty"`
}

// NormalizedTrackRecord models a source-cleaned track before cross-source merge.
type NormalizedTrackRecord struct {
	Source                     string `json:"source"`
	SourceTrackID              string `json:"source_track_id,omitempty"`
	Name                       string `json:"name"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	ReleaseName                string `json:"release_name,omitempty"`
	ReleaseCanonicalSlug       string `json:"release_canonical_slug,omitempty"`
	SpotifyURL                 string `json:"spotify_url,omitempty"`
	MusicBrainzTrackID         string `json:"musicbrainz_track_id,omitempty"`
}

// NormalizedGenreRecord models one source genre mapped into the garden taxonomy.
type NormalizedGenreRecord struct {
	Source             string `json:"source"`
	SourceGenre        string `json:"source_genre"`
	CanonicalGenreSlug string `json:"canonical_genre_slug,omitempty"`
}

// AggregatedArtistRecord is the persisted canonical artist record.
type AggregatedArtistRecord struct {
	CanonicalSlug       string               `json:"canonical_slug"`
	Name                string               `json:"name"`
	SpotifyArtistID     string               `json:"spotify_artist_id,omitempty"`
	MusicBrainzArtistID string               `json:"musicbrainz_artist_id,omitempty"`
	SpotifyURL          string               `json:"spotify_url,omitempty"`
	Genres              []string             `json:"genres,omitempty"`
	SourceGenres        []string             `json:"source_genres,omitempty"`
	Images              []models.ArtistImage `json:"images,omitempty"`
	LastUpdated         string               `json:"last_updated,omitempty"`
}

// AggregatedReleaseRecord is the persisted canonical release record.
type AggregatedReleaseRecord struct {
	CanonicalSlug              string `json:"canonical_slug"`
	Name                       string `json:"name"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	SpotifyAlbumID             string `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseID       string `json:"musicbrainz_release_id,omitempty"`
	MusicBrainzReleaseGroupID  string `json:"musicbrainz_release_group_id,omitempty"`
	LastUpdated                string `json:"last_updated,omitempty"`
}

// AggregatedTrackRecord is the persisted canonical track record.
type AggregatedTrackRecord struct {
	CanonicalSlug              string `json:"canonical_slug"`
	Name                       string `json:"name"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	ReleaseCanonicalSlug       string `json:"release_canonical_slug,omitempty"`
	ReleaseName                string `json:"release_name,omitempty"`
	SpotifyTrackID             string `json:"spotify_track_id,omitempty"`
	MusicBrainzTrackID         string `json:"musicbrainz_track_id,omitempty"`
	SpotifyURL                 string `json:"spotify_url,omitempty"`
	LastUpdated                string `json:"last_updated,omitempty"`
}

// AggregatedGenreRecord is the persisted canonical genre record.
type AggregatedGenreRecord struct {
	CanonicalSlug string   `json:"canonical_slug"`
	Aliases       []string `json:"aliases,omitempty"`
	Pending       bool     `json:"pending"`
}

// RawFetchManifest records how a raw source payload was fetched.
type RawFetchManifest struct {
	Source      string `json:"source"`
	Endpoint    string `json:"endpoint"`
	RequestURL  string `json:"request_url"`
	CacheKey    string `json:"cache_key,omitempty"`
	FetchedAt   string `json:"fetched_at"`
	Status      string `json:"status,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// EnsureLayout creates the raw/normalized/aggregated directory layout.
func EnsureLayout(dataRoot string) error {
	dirs := []string{
		filepath.Join(dataRoot, "raw", "spotify", "recently-played"),
		filepath.Join(dataRoot, "raw", "spotify", "artists"),
		filepath.Join(dataRoot, "raw", "spotify", "top-artists"),
		filepath.Join(dataRoot, "raw", "musicbrainz", "artist-search"),
		filepath.Join(dataRoot, "raw", "musicbrainz", "artists"),
		filepath.Join(dataRoot, "raw", "musicbrainz", "release-group-search"),
		filepath.Join(dataRoot, "raw", "musicbrainz", "release-groups"),
		filepath.Join(dataRoot, "raw", "musicbrainz", "releases"),
		filepath.Join(dataRoot, "raw", "wikipedia"),
		filepath.Join(dataRoot, "normalized", "artists"),
		filepath.Join(dataRoot, "normalized", "releases"),
		filepath.Join(dataRoot, "normalized", "tracks"),
		filepath.Join(dataRoot, "normalized", "genres"),
		filepath.Join(dataRoot, "aggregated", "artists"),
		filepath.Join(dataRoot, "aggregated", "releases"),
		filepath.Join(dataRoot, "aggregated", "tracks"),
		filepath.Join(dataRoot, "aggregated", "genres"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return nil
}

// WriteRawSpotifyRecentlyPlayed stores an unchanged Spotify recently-played response body.
func WriteRawSpotifyRecentlyPlayed(dataRoot string, fetchedAt time.Time, body []byte) (string, error) {
	return writeRaw(filepathJoinRaw(dataRoot, "spotify", "recently-played", fetchedAt, "recently-played"), body)
}

// WriteRawSpotifyArtists stores an unchanged Spotify artists batch response body.
func WriteRawSpotifyArtists(dataRoot string, fetchedAt time.Time, batch int, body []byte) (string, error) {
	stem := fmt.Sprintf("artists-batch-%02d", batch)
	return writeRaw(filepathJoinRaw(dataRoot, "spotify", "artists", fetchedAt, stem), body)
}

// WriteRawSpotifyTopArtists stores an unchanged Spotify top-artists response body.
func WriteRawSpotifyTopArtists(dataRoot, timeRange string, fetchedAt time.Time, body []byte) (string, error) {
	stem := "top-artists-" + strings.ReplaceAll(timeRange, "_", "-")
	return writeRaw(filepathJoinRaw(dataRoot, "spotify", "top-artists", fetchedAt, stem), body)
}

// WriteRawMusicBrainzResponse stores a deterministic MusicBrainz payload and manifest.
func WriteRawMusicBrainzResponse(dataRoot, kind, stem string, manifest RawFetchManifest, body []byte) (string, string, error) {
	dir := filepath.Join(dataRoot, "raw", "musicbrainz", kind)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}
	payloadPath := filepath.Join(dir, stem+".json")
	manifestPath := filepath.Join(dir, stem+".manifest.json")
	if err := os.WriteFile(payloadPath, body, 0644); err != nil {
		return "", "", err
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return "", "", err
	}
	return payloadPath, manifestPath, nil
}

// WriteNormalizedMusicBrainzArtist stores a normalized MusicBrainz artist record.
func WriteNormalizedMusicBrainzArtist(dataRoot, slug string, payload NormalizedArtistRecord) error {
	path := filepath.Join(dataRoot, "normalized", "artists", "musicbrainz--"+slug+".json")
	return writeJSON(path, payload)
}

// WriteNormalizedMusicBrainzRelease stores a normalized MusicBrainz release record.
func WriteNormalizedMusicBrainzRelease(dataRoot, slug string, payload NormalizedReleaseRecord) error {
	path := filepath.Join(dataRoot, "normalized", "releases", "musicbrainz--"+slug+".json")
	return writeJSON(path, payload)
}

// WriteNormalizedMusicBrainzGenres stores normalized MusicBrainz genre records.
func WriteNormalizedMusicBrainzGenres(dataRoot string, records []NormalizedGenreRecord) error {
	dir := filepath.Join(dataRoot, "normalized", "genres")
	for _, record := range records {
		stem := genres.Slug("musicbrainz-" + record.SourceGenre)
		if stem == "" {
			continue
		}
		if err := writeJSON(filepath.Join(dir, stem+".json"), record); err != nil {
			return err
		}
	}
	return nil
}

func writeRaw(path string, body []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func filepathJoinRaw(dataRoot, source, kind string, fetchedAt time.Time, stem string) string {
	ts := fetchedAt.UTC().Format("2006-01-02T15-04-05Z")
	return filepath.Join(dataRoot, "raw", source, kind, fmt.Sprintf("%s-%s.json", ts, stem))
}

// SyncAggregatedStore rewrites aggregated artist/release/genre records from the canonical store.
func SyncAggregatedStore(dataRoot string, store *genres.Store) error {
	if err := EnsureLayout(dataRoot); err != nil {
		return err
	}
	if err := SyncNormalizedStore(dataRoot, store); err != nil {
		return err
	}
	if err := writeAggregatedArtists(filepath.Join(dataRoot, "aggregated", "artists"), store); err != nil {
		return err
	}
	if err := writeAggregatedReleases(filepath.Join(dataRoot, "aggregated", "releases"), store); err != nil {
		return err
	}
	if err := writeAggregatedTracks(filepath.Join(dataRoot, "aggregated", "tracks"), store); err != nil {
		return err
	}
	if err := writeAggregatedGenres(filepath.Join(dataRoot, "aggregated", "genres"), store); err != nil {
		return err
	}
	return nil
}

// SyncNormalizedStore rewrites normalized artist/release/track/genre records
// for the currently supported source adapters.
func SyncNormalizedStore(dataRoot string, store *genres.Store) error {
	if err := EnsureLayout(dataRoot); err != nil {
		return err
	}
	if err := writeNormalizedArtists(filepath.Join(dataRoot, "normalized", "artists"), store); err != nil {
		return err
	}
	if err := writeNormalizedReleases(filepath.Join(dataRoot, "normalized", "releases"), store); err != nil {
		return err
	}
	if err := writeNormalizedTracks(filepath.Join(dataRoot, "normalized", "tracks"), store); err != nil {
		return err
	}
	if err := writeNormalizedGenres(filepath.Join(dataRoot, "normalized", "genres"), store); err != nil {
		return err
	}
	return nil
}

func writeNormalizedArtists(dir string, store *genres.Store) error {
	for _, record := range genres.ArtistRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := NormalizedArtistRecord{
			Source:              "spotify",
			SourceArtistID:      record.SpotifyArtistID,
			Name:                record.Name,
			SpotifyURL:          record.SpotifyURL,
			MusicBrainzArtistID: record.MusicBrainzArtistID,
			SourceGenres:        record.SourceGenres,
			CanonicalGenreSlugs: record.Genres,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeNormalizedReleases(dir string, store *genres.Store) error {
	for _, record := range genres.ReleaseRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := NormalizedReleaseRecord{
			Source:                     "spotify",
			SourceReleaseID:            record.SpotifyAlbumID,
			Name:                       record.Name,
			PrimaryArtistName:          record.PrimaryArtistName,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			MusicBrainzReleaseID:       record.MusicBrainzReleaseID,
			MusicBrainzReleaseGroupID:  record.MusicBrainzReleaseGroupID,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeNormalizedTracks(dir string, store *genres.Store) error {
	for _, record := range genres.TrackRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := NormalizedTrackRecord{
			Source:                     "spotify",
			SourceTrackID:              record.SpotifyTrackID,
			Name:                       record.Name,
			PrimaryArtistName:          record.PrimaryArtistName,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			ReleaseName:                record.ReleaseName,
			ReleaseCanonicalSlug:       record.ReleaseSlug,
			SpotifyURL:                 record.SpotifyURL,
			MusicBrainzTrackID:         record.MusicBrainzTrackID,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeNormalizedGenres(dir string, store *genres.Store) error {
	for alias, canonical := range store.GenreAliases {
		name := genres.Slug("spotify-" + alias)
		payload := NormalizedGenreRecord{
			Source:             "spotify",
			SourceGenre:        alias,
			CanonicalGenreSlug: canonical,
		}
		if err := writeJSON(filepath.Join(dir, name+".json"), payload); err != nil {
			return err
		}
	}
	for _, pending := range store.PendingGenreAliases {
		name := genres.Slug("spotify-" + pending)
		payload := NormalizedGenreRecord{
			Source:      "spotify",
			SourceGenre: pending,
		}
		if err := writeJSON(filepath.Join(dir, name+".json"), payload); err != nil {
			return err
		}
	}
	return nil
}

func writeAggregatedArtists(dir string, store *genres.Store) error {
	for _, record := range genres.ArtistRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := AggregatedArtistRecord{
			CanonicalSlug:       record.Slug,
			Name:                record.Name,
			SpotifyArtistID:     record.SpotifyArtistID,
			MusicBrainzArtistID: record.MusicBrainzArtistID,
			SpotifyURL:          record.SpotifyURL,
			Genres:              record.Genres,
			SourceGenres:        record.SourceGenres,
			Images:              record.Images,
			LastUpdated:         record.LastUpdated,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeAggregatedReleases(dir string, store *genres.Store) error {
	for _, record := range genres.ReleaseRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := AggregatedReleaseRecord{
			CanonicalSlug:              record.Slug,
			Name:                       record.Name,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			PrimaryArtistName:          record.PrimaryArtistName,
			SpotifyAlbumID:             record.SpotifyAlbumID,
			MusicBrainzReleaseID:       record.MusicBrainzReleaseID,
			MusicBrainzReleaseGroupID:  record.MusicBrainzReleaseGroupID,
			LastUpdated:                record.LastUpdated,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeAggregatedTracks(dir string, store *genres.Store) error {
	for _, record := range genres.TrackRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := AggregatedTrackRecord{
			CanonicalSlug:              record.Slug,
			Name:                       record.Name,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			PrimaryArtistName:          record.PrimaryArtistName,
			ReleaseCanonicalSlug:       record.ReleaseSlug,
			ReleaseName:                record.ReleaseName,
			SpotifyTrackID:             record.SpotifyTrackID,
			MusicBrainzTrackID:         record.MusicBrainzTrackID,
			SpotifyURL:                 record.SpotifyURL,
			LastUpdated:                record.LastUpdated,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeAggregatedGenres(dir string, store *genres.Store) error {
	aliasBuckets := map[string][]string{}
	for alias, canonical := range store.GenreAliases {
		aliasBuckets[canonical] = append(aliasBuckets[canonical], alias)
	}

	seenPending := map[string]bool{}
	for _, pending := range store.PendingGenreAliases {
		slug := genres.Slug(pending)
		if slug == "" {
			continue
		}
		if _, ok := aliasBuckets[slug]; !ok {
			aliasBuckets[slug] = nil
		}
		seenPending[slug] = true
	}

	slugs := make([]string, 0, len(aliasBuckets))
	for slug := range aliasBuckets {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	for _, slug := range slugs {
		aliases := dedupeStrings(aliasBuckets[slug])
		sort.Strings(aliases)
		payload := AggregatedGenreRecord{
			CanonicalSlug: slug,
			Aliases:       aliases,
			Pending:       seenPending[slug],
		}
		if err := writeJSON(filepath.Join(dir, slug+".json"), payload); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
