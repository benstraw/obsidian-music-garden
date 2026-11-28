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
	Source              string                         `json:"source"`
	SourceArtistID      string                         `json:"source_artist_id,omitempty"`
	Name                string                         `json:"name"`
	SpotifyURL          string                         `json:"spotify_url,omitempty"`
	MusicBrainzArtistID string                         `json:"musicbrainz_artist_id,omitempty"`
	SourceGenres        []string                       `json:"source_genres,omitempty"`
	CanonicalGenreSlugs []string                       `json:"canonical_genre_slugs,omitempty"`
	Status              string                         `json:"status,omitempty"`
	PageTitle           string                         `json:"page_title,omitempty"`
	Summary             string                         `json:"summary,omitempty"`
	CanonicalURL        string                         `json:"canonical_url,omitempty"`
	Candidates          []string                       `json:"candidates,omitempty"`
	Attribution         *genres.GenreSourceAttribution `json:"attribution,omitempty"`
	Image               *genres.GenreImageRecord       `json:"image,omitempty"`
	ImageCandidates     []genres.GenreImageRecord      `json:"image_candidates,omitempty"`
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
	Source                     string   `json:"source"`
	SourceTrackID              string   `json:"source_track_id,omitempty"`
	Name                       string   `json:"name"`
	PrimaryArtistName          string   `json:"primary_artist_name,omitempty"`
	PrimaryArtistCanonicalSlug string   `json:"primary_artist_canonical_slug,omitempty"`
	AdditionalArtistSlugs      []string `json:"additional_artist_slugs,omitempty"`
	AlbumArtistSlugs           []string `json:"album_artist_slugs,omitempty"`
	ReleaseName                string   `json:"release_name,omitempty"`
	ReleaseCanonicalSlug       string   `json:"release_canonical_slug,omitempty"`
	SpotifyURL                 string   `json:"spotify_url,omitempty"`
	MusicBrainzTrackID         string   `json:"musicbrainz_track_id,omitempty"`
	LegacyPlayCount            int      `json:"legacy_play_count,omitempty"`
}

// NormalizedGenreRecord models one source genre mapped into the garden taxonomy.
type NormalizedGenreRecord struct {
	Source             string                         `json:"source"`
	SourceGenre        string                         `json:"source_genre"`
	CanonicalGenreSlug string                         `json:"canonical_genre_slug,omitempty"`
	Status             string                         `json:"status,omitempty"`
	PageTitle          string                         `json:"page_title,omitempty"`
	Summary            string                         `json:"summary,omitempty"`
	CanonicalURL       string                         `json:"canonical_url,omitempty"`
	Candidates         []string                       `json:"candidates,omitempty"`
	Attribution        *genres.GenreSourceAttribution `json:"attribution,omitempty"`
	Image              *genres.GenreImageRecord       `json:"image,omitempty"`
	ImageCandidates    []genres.GenreImageRecord      `json:"image_candidates,omitempty"`
}

// AggregatedArtistRecord is the persisted canonical artist record.
type AggregatedArtistRecord struct {
	CanonicalSlug       string                         `json:"canonical_slug"`
	Name                string                         `json:"name"`
	PlayCount           int                            `json:"play_count,omitempty"`
	LegacyPlayCount     int                            `json:"legacy_play_count,omitempty"`
	SpotifyArtistID     string                         `json:"spotify_artist_id,omitempty"`
	MusicBrainzArtistID string                         `json:"musicbrainz_artist_id,omitempty"`
	SpotifyURL          string                         `json:"spotify_url,omitempty"`
	Genres              []string                       `json:"genres,omitempty"`
	SourceGenres        []string                       `json:"source_genres,omitempty"`
	Images              []models.ArtistImage           `json:"images,omitempty"`
	Status              string                         `json:"status,omitempty"`
	WikipediaTitle      string                         `json:"wikipedia_title,omitempty"`
	WikipediaURL        string                         `json:"wikipedia_url,omitempty"`
	Summary             string                         `json:"summary,omitempty"`
	Candidates          []string                       `json:"candidates,omitempty"`
	Attribution         *genres.GenreSourceAttribution `json:"attribution,omitempty"`
	Image               *genres.GenreImageRecord       `json:"image,omitempty"`
	ImageCandidates     []genres.GenreImageRecord      `json:"image_candidates,omitempty"`
	LastUpdated         string                         `json:"last_updated,omitempty"`
}

// AggregatedReleaseRecord is the persisted canonical release record.
type AggregatedReleaseRecord struct {
	CanonicalSlug              string `json:"canonical_slug"`
	Name                       string `json:"name"`
	PlayCount                  int    `json:"play_count,omitempty"`
	LegacyPlayCount            int    `json:"legacy_play_count,omitempty"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	SpotifyAlbumID             string `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseID       string `json:"musicbrainz_release_id,omitempty"`
	MusicBrainzReleaseGroupID  string `json:"musicbrainz_release_group_id,omitempty"`
	LastUpdated                string `json:"last_updated,omitempty"`
}

// AggregatedTrackRecord is the persisted canonical track record.
type AggregatedTrackRecord struct {
	CanonicalSlug              string   `json:"canonical_slug"`
	Name                       string   `json:"name"`
	PlayCount                  int      `json:"play_count,omitempty"`
	PrimaryArtistCanonicalSlug string   `json:"primary_artist_canonical_slug,omitempty"`
	PrimaryArtistName          string   `json:"primary_artist_name,omitempty"`
	AdditionalArtistSlugs      []string `json:"additional_artist_slugs,omitempty"`
	AlbumArtistSlugs           []string `json:"album_artist_slugs,omitempty"`
	ReleaseCanonicalSlug       string   `json:"release_canonical_slug,omitempty"`
	ReleaseName                string   `json:"release_name,omitempty"`
	SpotifyTrackID             string   `json:"spotify_track_id,omitempty"`
	MusicBrainzTrackID         string   `json:"musicbrainz_track_id,omitempty"`
	SpotifyURL                 string   `json:"spotify_url,omitempty"`
	LegacyPlayCount            int      `json:"legacy_play_count,omitempty"`
	LastUpdated                string   `json:"last_updated,omitempty"`
}

// AggregatedSourceReference documents where a major field in the aggregate came from.
type AggregatedSourceReference struct {
	Role        string `json:"role"`
	Source      string `json:"source"`
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"`
	RetrievedAt string `json:"retrieved_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Note        string `json:"note,omitempty"`
}

// GenreListeningStats summarizes local listening history for one canonical genre.
type GenreListeningStats struct {
	Source             string `json:"source"`
	PlayCount          int    `json:"play_count"`
	UniqueArtistCount  int    `json:"unique_artist_count"`
	UniqueReleaseCount int    `json:"unique_release_count"`
	UniqueTrackCount   int    `json:"unique_track_count"`
	FirstPlayedAt      string `json:"first_played_at,omitempty"`
	LastPlayedAt       string `json:"last_played_at,omitempty"`
}

// AggregatedGenreArtist summarizes one artist within a genre aggregate.
type AggregatedGenreArtist struct {
	CanonicalSlug       string `json:"canonical_slug"`
	Name                string `json:"name"`
	PlayCount           int    `json:"play_count"`
	SpotifyArtistID     string `json:"spotify_artist_id,omitempty"`
	MusicBrainzArtistID string `json:"musicbrainz_artist_id,omitempty"`
	SpotifyURL          string `json:"spotify_url,omitempty"`
}

// AggregatedGenreRelease summarizes one release within a genre aggregate.
type AggregatedGenreRelease struct {
	CanonicalSlug              string `json:"canonical_slug"`
	Name                       string `json:"name"`
	PlayCount                  int    `json:"play_count"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	SpotifyAlbumID             string `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseID       string `json:"musicbrainz_release_id,omitempty"`
	MusicBrainzReleaseGroupID  string `json:"musicbrainz_release_group_id,omitempty"`
}

// AggregatedGenreTrack summarizes one track within a genre aggregate.
type AggregatedGenreTrack struct {
	CanonicalSlug              string `json:"canonical_slug"`
	Name                       string `json:"name"`
	PlayCount                  int    `json:"play_count"`
	PrimaryArtistCanonicalSlug string `json:"primary_artist_canonical_slug,omitempty"`
	PrimaryArtistName          string `json:"primary_artist_name,omitempty"`
	ReleaseCanonicalSlug       string `json:"release_canonical_slug,omitempty"`
	ReleaseName                string `json:"release_name,omitempty"`
	SpotifyTrackID             string `json:"spotify_track_id,omitempty"`
	MusicBrainzTrackID         string `json:"musicbrainz_track_id,omitempty"`
	SpotifyURL                 string `json:"spotify_url,omitempty"`
}

// AggregatedGenreRecord is the persisted canonical genre record.
type AggregatedGenreRecord struct {
	CanonicalSlug   string                         `json:"canonical_slug"`
	DisplayTitle    string                         `json:"display_title,omitempty"`
	DisplayName     string                         `json:"display_name,omitempty"`
	ParentSlug      string                         `json:"parent_slug,omitempty"`
	Notes           string                         `json:"notes,omitempty"`
	Aliases         []string                       `json:"aliases,omitempty"`
	Pending         bool                           `json:"pending"`
	WorkflowState   string                         `json:"workflow_state,omitempty"`
	Status          string                         `json:"status,omitempty"`
	WikipediaTitle  string                         `json:"wikipedia_title,omitempty"`
	WikipediaURL    string                         `json:"wikipedia_url,omitempty"`
	Summary         string                         `json:"summary,omitempty"`
	Candidates      []string                       `json:"candidates,omitempty"`
	Attribution     *genres.GenreSourceAttribution `json:"attribution,omitempty"`
	Image           *genres.GenreImageRecord       `json:"image,omitempty"`
	ImageCandidates []genres.GenreImageRecord      `json:"image_candidates,omitempty"`
	ListeningStats  GenreListeningStats            `json:"listening_stats"`
	TopArtists      []AggregatedGenreArtist        `json:"top_artists,omitempty"`
	TopReleases     []AggregatedGenreRelease       `json:"top_releases,omitempty"`
	TopTracks       []AggregatedGenreTrack         `json:"top_tracks,omitempty"`
	SourceRefs      []AggregatedSourceReference    `json:"source_refs,omitempty"`
	LastUpdated     string                         `json:"last_updated,omitempty"`
}

type derivedTrackRecord struct {
	Slug                  string
	Name                  string
	PlayCount             int
	PrimaryArtistSlug     string
	PrimaryArtistName     string
	AdditionalArtistSlugs []string
	ReleaseSlug           string
	ReleaseName           string
	SpotifyTrackID        string
	MusicBrainzTrackID    string
	SpotifyURL            string
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
		filepath.Join(dataRoot, "raw", "wikipedia", "search"),
		filepath.Join(dataRoot, "raw", "wikipedia", "summaries"),
		filepath.Join(dataRoot, "raw", "wikipedia", "page-images"),
		filepath.Join(dataRoot, "raw", "wikipedia", "commons-images"),
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

// WriteRawWikipediaResponse stores a deterministic Wikipedia/Wikimedia payload and manifest.
func WriteRawWikipediaResponse(dataRoot, kind, stem string, manifest RawFetchManifest, body []byte) (string, string, error) {
	dir := filepath.Join(dataRoot, "raw", "wikipedia", kind)
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

// WriteNormalizedWikipediaGenre stores a normalized editorial genre record.
func WriteNormalizedWikipediaGenre(dataRoot, slug string, payload NormalizedGenreRecord) error {
	path := filepath.Join(dataRoot, "normalized", "genres", "wikipedia--"+slug+".json")
	return writeJSON(path, payload)
}

// WriteNormalizedWikipediaArtist stores a normalized editorial artist record.
func WriteNormalizedWikipediaArtist(dataRoot, slug string, payload NormalizedArtistRecord) error {
	path := filepath.Join(dataRoot, "normalized", "artists", "wikipedia--"+slug+".json")
	return writeJSON(path, payload)
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

// SyncAggregatedStore rewrites aggregated artist/release/track/genre records from the canonical store.
func SyncAggregatedStore(dataRoot string, store *genres.Store, plays []models.Play) error {
	if err := EnsureLayout(dataRoot); err != nil {
		return err
	}
	if err := SyncNormalizedStore(dataRoot, store, plays); err != nil {
		return err
	}
	if err := writeAggregatedArtists(filepath.Join(dataRoot, "aggregated", "artists"), store, plays); err != nil {
		return err
	}
	if err := writeAggregatedReleases(filepath.Join(dataRoot, "aggregated", "releases"), store, plays); err != nil {
		return err
	}
	if err := writeAggregatedTracks(filepath.Join(dataRoot, "aggregated", "tracks"), store, plays); err != nil {
		return err
	}
	if err := writeAggregatedGenres(filepath.Join(dataRoot, "aggregated", "genres"), dataRoot, store, plays); err != nil {
		return err
	}
	return nil
}

// SyncNormalizedStore rewrites normalized artist/release/track/genre records
// for the currently supported source adapters.
func SyncNormalizedStore(dataRoot string, store *genres.Store, plays []models.Play) error {
	if err := EnsureLayout(dataRoot); err != nil {
		return err
	}
	if err := writeNormalizedArtists(filepath.Join(dataRoot, "normalized", "artists"), store); err != nil {
		return err
	}
	if err := writeNormalizedReleases(filepath.Join(dataRoot, "normalized", "releases"), store); err != nil {
		return err
	}
	if err := writeNormalizedTracks(filepath.Join(dataRoot, "normalized", "tracks"), store, plays); err != nil {
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
			Status:              record.Status,
			PageTitle:           record.WikipediaTitle,
			Summary:             record.Summary,
			CanonicalURL:        record.WikipediaURL,
			Candidates:          append([]string(nil), record.Candidates...),
			Attribution:         record.Attribution,
			Image:               record.Image,
			ImageCandidates:     append([]genres.GenreImageRecord(nil), record.ImageCandidates...),
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

func writeNormalizedTracks(dir string, store *genres.Store, plays []models.Play) error {
	for _, record := range derivedTrackRecords(store, plays) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := NormalizedTrackRecord{
			Source:                     "spotify",
			SourceTrackID:              record.SpotifyTrackID,
			Name:                       record.Name,
			PrimaryArtistName:          record.PrimaryArtistName,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			AdditionalArtistSlugs:      append([]string(nil), record.AdditionalArtistSlugs...),
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

func writeAggregatedArtists(dir string, store *genres.Store, plays []models.Play) error {
	observedArtistCounts, legacyArtistCounts, _, _ := aggregateEntityCounts(store, plays)
	for _, record := range genres.ArtistRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := AggregatedArtistRecord{
			CanonicalSlug:       record.Slug,
			Name:                record.Name,
			PlayCount:           observedArtistCounts[record.Slug] + legacyArtistCounts[record.Slug],
			LegacyPlayCount:     legacyArtistCounts[record.Slug],
			SpotifyArtistID:     record.SpotifyArtistID,
			MusicBrainzArtistID: record.MusicBrainzArtistID,
			SpotifyURL:          record.SpotifyURL,
			Genres:              record.Genres,
			SourceGenres:        record.SourceGenres,
			Images:              record.Images,
			Status:              record.Status,
			WikipediaTitle:      record.WikipediaTitle,
			WikipediaURL:        record.WikipediaURL,
			Summary:             record.Summary,
			Candidates:          append([]string(nil), record.Candidates...),
			Attribution:         record.Attribution,
			Image:               record.Image,
			ImageCandidates:     append([]genres.GenreImageRecord(nil), record.ImageCandidates...),
			LastUpdated:         record.LastUpdated,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeAggregatedReleases(dir string, store *genres.Store, plays []models.Play) error {
	_, _, observedReleaseCounts, legacyReleaseCounts := aggregateEntityCounts(store, plays)
	for _, record := range genres.ReleaseRecords(store) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := AggregatedReleaseRecord{
			CanonicalSlug:              record.Slug,
			Name:                       record.Name,
			PlayCount:                  observedReleaseCounts[record.Slug] + legacyReleaseCounts[record.Slug],
			LegacyPlayCount:            legacyReleaseCounts[record.Slug],
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

func writeAggregatedTracks(dir string, store *genres.Store, plays []models.Play) error {
	for _, record := range derivedTrackRecords(store, plays) {
		path := filepath.Join(dir, record.Slug+".json")
		payload := AggregatedTrackRecord{
			CanonicalSlug:              record.Slug,
			Name:                       record.Name,
			PlayCount:                  record.PlayCount,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			PrimaryArtistName:          record.PrimaryArtistName,
			AdditionalArtistSlugs:      append([]string(nil), record.AdditionalArtistSlugs...),
			ReleaseCanonicalSlug:       record.ReleaseSlug,
			ReleaseName:                record.ReleaseName,
			SpotifyTrackID:             record.SpotifyTrackID,
			MusicBrainzTrackID:         record.MusicBrainzTrackID,
			SpotifyURL:                 record.SpotifyURL,
		}
		if err := writeJSON(path, payload); err != nil {
			return err
		}
	}
	return nil
}

func writeAggregatedGenres(dir, dataRoot string, store *genres.Store, plays []models.Play) error {
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
		aliases, ok := aliasBuckets[slug]
		if !ok {
			aliasBuckets[slug] = nil
			seenPending[slug] = true
			continue
		}
		if len(aliases) == 0 {
			seenPending[slug] = true
		}
	}

	slugs := make([]string, 0, len(aliasBuckets))
	for slug := range aliasBuckets {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	for _, slug := range slugs {
		payload, err := BuildAggregatedGenreRecord(dataRoot, store, plays, slug, aliasBuckets[slug], seenPending[slug])
		if err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(dir, slug+".json"), payload); err != nil {
			return err
		}
	}
	return nil
}

// BuildAggregatedGenreRecord merges taxonomy, editorial data, and local listening data
// into one deterministic canonical genre record.
func BuildAggregatedGenreRecord(dataRoot string, store *genres.Store, plays []models.Play, slug string, aliases []string, pending bool) (AggregatedGenreRecord, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return AggregatedGenreRecord{}, fmt.Errorf("empty genre slug")
	}

	aliases = dedupeStrings(aliases)
	sort.Strings(aliases)
	payload := AggregatedGenreRecord{
		CanonicalSlug: slug,
		DisplayTitle:  humanizeSlug(slug),
		DisplayName:   humanizeSlug(slug),
		Aliases:       aliases,
		Pending:       pending,
		WorkflowState: genres.WorkflowStateDraft,
		ListeningStats: GenreListeningStats{
			Source: "spotify-local-plays",
		},
	}

	if record, ok := genres.GenreEditorial(store, slug); ok {
		payload.DisplayTitle = firstNonEmpty(record.DisplayName, payload.DisplayTitle)
		payload.DisplayName = firstNonEmpty(record.DisplayName, payload.DisplayName)
		payload.ParentSlug = record.ParentSlug
		payload.Notes = record.Notes
		payload.WorkflowState = genres.GenreWorkflowState(record)
		payload.Status = record.Status
		payload.WikipediaTitle = record.WikipediaTitle
		payload.WikipediaURL = record.WikipediaURL
		payload.Summary = record.Summary
		payload.Candidates = append([]string(nil), record.Candidates...)
		payload.Attribution = record.Attribution
		payload.Image = record.Image
		payload.ImageCandidates = append([]genres.GenreImageRecord(nil), record.ImageCandidates...)
		payload.LastUpdated = record.LastUpdated
	}

	artistCounts := map[string]int{}
	releaseCounts := map[string]int{}
	trackCounts := map[string]int{}
	trackRecords := map[string]derivedTrackRecord{}
	seenArtists := map[string]bool{}
	seenReleases := map[string]bool{}
	seenTracks := map[string]bool{}

	for _, play := range plays {
		artist, ok := artistRecordForPlay(store, play)
		if !ok || !containsString(artist.Genres, slug) {
			continue
		}
		payload.ListeningStats.PlayCount++
		if payload.ListeningStats.FirstPlayedAt == "" || play.PlayedAt < payload.ListeningStats.FirstPlayedAt {
			payload.ListeningStats.FirstPlayedAt = play.PlayedAt
		}
		if payload.ListeningStats.LastPlayedAt == "" || play.PlayedAt > payload.ListeningStats.LastPlayedAt {
			payload.ListeningStats.LastPlayedAt = play.PlayedAt
		}

		if artist.Slug != "" {
			artistCounts[artist.Slug]++
			seenArtists[artist.Slug] = true
		}

		releaseSlug := firstNonEmpty(play.ReleaseSlug, releaseSlugForPlay(store, play))
		if releaseSlug != "" {
			releaseCounts[releaseSlug]++
			seenReleases[releaseSlug] = true
		}

		trackSlug := firstNonEmpty(play.TrackSlug, trackSlugForPlay(store, play))
		if trackSlug != "" {
			trackCounts[trackSlug]++
			seenTracks[trackSlug] = true
			if track, ok := derivedTrackRecordForPlay(store, play); ok {
				trackRecords[trackSlug] = mergeDerivedTrackRecord(trackRecords[trackSlug], track)
			}
		}
	}

	payload.ListeningStats.UniqueArtistCount = len(seenArtists)
	payload.ListeningStats.UniqueReleaseCount = len(seenReleases)
	payload.ListeningStats.UniqueTrackCount = len(seenTracks)
	payload.TopArtists = topGenreArtists(store, artistCounts, 10)
	payload.TopReleases = topGenreReleases(store, releaseCounts, 10)
	payload.TopTracks = topGenreTracks(trackRecords, trackCounts, 10)
	payload.SourceRefs = genreSourceRefs(dataRoot, payload, len(aliases) > 0 || payload.DisplayTitle != "" || payload.ParentSlug != "" || payload.Notes != "")
	payload.LastUpdated = maxTimestamp(payload.LastUpdated, payload.ListeningStats.LastPlayedAt)
	return payload, nil
}

// RebuildAggregatedGenre rewrites one canonical genre record from upstream data.
func RebuildAggregatedGenre(dataRoot string, store *genres.Store, plays []models.Play, slug string) (string, error) {
	if err := EnsureLayout(dataRoot); err != nil {
		return "", err
	}
	aliases, pending, ok := genreAliasesForSlug(store, slug)
	if !ok {
		return "", fmt.Errorf("unknown canonical genre slug %q", slug)
	}
	payload, err := BuildAggregatedGenreRecord(dataRoot, store, plays, slug, aliases, pending)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dataRoot, "aggregated", "genres", slug+".json")
	if err := writeJSON(path, payload); err != nil {
		return "", err
	}
	return path, nil
}

// RebuildAllAggregatedGenres rewrites every known canonical genre record.
func RebuildAllAggregatedGenres(dataRoot string, store *genres.Store, plays []models.Play) (int, error) {
	if err := EnsureLayout(dataRoot); err != nil {
		return 0, err
	}
	count := 0
	for _, slug := range knownGenreSlugs(store) {
		if _, err := RebuildAggregatedGenre(dataRoot, store, plays, slug); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadAggregatedGenre reads one aggregated genre record from disk.
func LoadAggregatedGenre(path string) (AggregatedGenreRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AggregatedGenreRecord{}, err
	}
	var record AggregatedGenreRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return AggregatedGenreRecord{}, err
	}
	return record, nil
}

// LoadAggregatedGenres reads all aggregated genre records from dir in slug order.
func LoadAggregatedGenres(dir string) ([]AggregatedGenreRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	records := make([]AggregatedGenreRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, err := LoadAggregatedGenre(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CanonicalSlug < records[j].CanonicalSlug
	})
	return records, nil
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

func aggregateEntityCounts(store *genres.Store, plays []models.Play) (map[string]int, map[string]int, map[string]int, map[string]int) {
	observedArtistCounts := map[string]int{}
	legacyArtistCounts := map[string]int{}
	observedReleaseCounts := map[string]int{}
	legacyReleaseCounts := map[string]int{}

	for _, play := range plays {
		if artist, ok := artistRecordForPlay(store, play); ok && artist.Slug != "" {
			observedArtistCounts[artist.Slug]++
		}
		if releaseSlug := firstNonEmpty(play.ReleaseSlug, releaseSlugForPlay(store, play)); releaseSlug != "" {
			observedReleaseCounts[releaseSlug]++
		}
	}

	return observedArtistCounts, legacyArtistCounts, observedReleaseCounts, legacyReleaseCounts
}

func aggregateTrackCounts(store *genres.Store, plays []models.Play) (map[string]int, map[string]int, map[string]int, map[string]int, map[string]int, map[string]int) {
	observedArtistCounts, legacyArtistCounts, observedReleaseCounts, legacyReleaseCounts := aggregateEntityCounts(store, plays)
	observedTrackCounts := map[string]int{}
	legacyTrackCounts := map[string]int{}

	for _, play := range plays {
		if trackSlug := firstNonEmpty(play.TrackSlug, trackSlugForPlay(store, play)); trackSlug != "" {
			observedTrackCounts[trackSlug]++
		}
	}

	return observedArtistCounts, legacyArtistCounts, observedReleaseCounts, legacyReleaseCounts, observedTrackCounts, legacyTrackCounts
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func genreAliasesForSlug(store *genres.Store, target string) ([]string, bool, bool) {
	aliases := []string{}
	for alias, canonical := range store.GenreAliases {
		if canonical == target {
			aliases = append(aliases, alias)
		}
	}
	pending := false
	if len(aliases) == 0 {
		for _, label := range store.PendingGenreAliases {
			if genres.Slug(label) == target {
				pending = true
				break
			}
		}
	}
	if len(aliases) == 0 {
		if _, ok := genres.GenreEditorial(store, target); !ok && !pending {
			return nil, false, false
		}
	}
	return dedupeStrings(aliases), pending, true
}

func knownGenreSlugs(store *genres.Store) []string {
	set := map[string]bool{}
	for _, slug := range store.GenreAliases {
		if slug != "" {
			set[slug] = true
		}
	}
	for _, label := range store.PendingGenreAliases {
		if slug := genres.Slug(label); slug != "" {
			set[slug] = true
		}
	}
	for slug := range store.GenreRecords {
		if slug != "" {
			set[slug] = true
		}
	}
	slugs := make([]string, 0, len(set))
	for slug := range set {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

func artistRecordForPlay(store *genres.Store, play models.Play) (genres.ArtistRecord, bool) {
	if play.ArtistSlug != "" {
		record, ok := store.Artists[genres.CanonicalArtistSlug(store, play.ArtistSlug)]
		if ok {
			return record, true
		}
	}
	if play.ArtistID != "" {
		if slug, ok := store.ArtistSourceIndex["spotify:"+play.ArtistID]; ok {
			record, ok := store.Artists[slug]
			return record, ok
		}
	}
	return genres.ArtistRecord{}, false
}

func releaseSlugForPlay(store *genres.Store, play models.Play) string {
	if play.ReleaseSlug != "" {
		return play.ReleaseSlug
	}
	if play.AlbumID != "" {
		if slug := store.ReleaseSourceIndex["spotify:"+play.AlbumID]; slug != "" {
			return slug
		}
	}
	if play.ArtistSlug != "" && play.AlbumName != "" {
		slug := genres.Slug(play.ArtistSlug + "--" + play.AlbumName)
		if _, ok := store.Releases[slug]; ok {
			return slug
		}
	}
	return ""
}

func trackSlugForPlay(store *genres.Store, play models.Play) string {
	if play.TrackSlug != "" {
		return play.TrackSlug
	}
	artistSlug := firstNonEmpty(play.ArtistSlug, artistSlugForPlay(store, play))
	if artistSlug != "" && play.TrackName != "" {
		return artistSlug + "--" + genres.Slug(play.TrackName)
	}
	if play.TrackName != "" {
		return genres.Slug(play.TrackName)
	}
	if play.TrackID != "" {
		return genres.Slug("track-" + play.TrackID)
	}
	return ""
}

func artistSlugForPlay(store *genres.Store, play models.Play) string {
	if play.ArtistSlug != "" {
		return genres.CanonicalArtistSlug(store, play.ArtistSlug)
	}
	if play.ArtistID != "" {
		if slug, ok := store.ArtistSourceIndex["spotify:"+play.ArtistID]; ok {
			return slug
		}
	}
	return ""
}

func derivedTrackRecordForPlay(store *genres.Store, play models.Play) (derivedTrackRecord, bool) {
	slug := trackSlugForPlay(store, play)
	if slug == "" {
		return derivedTrackRecord{}, false
	}
	artistSlug := firstNonEmpty(play.ArtistSlug, artistSlugForPlay(store, play))
	artistName := strings.TrimSpace(play.ArtistName)
	if artistName == "" && artistSlug != "" {
		if artist, ok := store.Artists[artistSlug]; ok {
			artistName = artist.Name
		}
	}
	additional := make([]string, 0, len(play.AdditionalArtists))
	for _, artist := range play.AdditionalArtists {
		if artist.ID == "" && artist.Name == "" {
			continue
		}
		record := genres.UpsertArtistMetadata(store, artist.ID, artist.Name, artist.SpotifyURL, "", nil, nil)
		if record.Slug != "" {
			additional = append(additional, record.Slug)
		}
	}
	return derivedTrackRecord{
		Slug:                  slug,
		Name:                  play.TrackName,
		PrimaryArtistSlug:     artistSlug,
		PrimaryArtistName:     artistName,
		AdditionalArtistSlugs: dedupeStrings(additional),
		ReleaseSlug:           firstNonEmpty(play.ReleaseSlug, releaseSlugForPlay(store, play)),
		ReleaseName:           play.AlbumName,
		SpotifyTrackID:        play.TrackID,
		MusicBrainzTrackID:    play.TrackMusicBrainzID,
		SpotifyURL:            play.TrackSpotifyURL,
	}, true
}

func mergeDerivedTrackRecord(existing, next derivedTrackRecord) derivedTrackRecord {
	if existing.Slug == "" {
		return next
	}
	existing.Name = firstNonEmpty(existing.Name, next.Name)
	existing.PrimaryArtistSlug = firstNonEmpty(existing.PrimaryArtistSlug, next.PrimaryArtistSlug)
	existing.PrimaryArtistName = firstNonEmpty(existing.PrimaryArtistName, next.PrimaryArtistName)
	existing.ReleaseSlug = firstNonEmpty(existing.ReleaseSlug, next.ReleaseSlug)
	existing.ReleaseName = firstNonEmpty(existing.ReleaseName, next.ReleaseName)
	existing.SpotifyTrackID = firstNonEmpty(existing.SpotifyTrackID, next.SpotifyTrackID)
	existing.MusicBrainzTrackID = firstNonEmpty(existing.MusicBrainzTrackID, next.MusicBrainzTrackID)
	existing.SpotifyURL = firstNonEmpty(existing.SpotifyURL, next.SpotifyURL)
	if len(next.AdditionalArtistSlugs) > 0 {
		existing.AdditionalArtistSlugs = dedupeStrings(append(existing.AdditionalArtistSlugs, next.AdditionalArtistSlugs...))
	}
	if next.PlayCount > existing.PlayCount {
		existing.PlayCount = next.PlayCount
	}
	return existing
}

func derivedTrackRecords(store *genres.Store, plays []models.Play) []derivedTrackRecord {
	recordsBySlug := map[string]derivedTrackRecord{}
	for _, play := range plays {
		record, ok := derivedTrackRecordForPlay(store, play)
		if !ok {
			continue
		}
		record.PlayCount = 1
		record = mergeDerivedTrackRecord(recordsBySlug[record.Slug], record)
		record.PlayCount = recordsBySlug[record.Slug].PlayCount + 1
		recordsBySlug[record.Slug] = record
	}
	records := make([]derivedTrackRecord, 0, len(recordsBySlug))
	for _, record := range recordsBySlug {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Name != records[j].Name {
			return records[i].Name < records[j].Name
		}
		return records[i].Slug < records[j].Slug
	})
	return records
}

func topGenreArtists(store *genres.Store, counts map[string]int, limit int) []AggregatedGenreArtist {
	type bucket struct {
		slug  string
		count int
		name  string
	}
	items := make([]bucket, 0, len(counts))
	for slug, count := range counts {
		record, ok := store.Artists[slug]
		if !ok {
			continue
		}
		items = append(items, bucket{slug: slug, count: count, name: record.Name})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		if items[i].name != items[j].name {
			return items[i].name < items[j].name
		}
		return items[i].slug < items[j].slug
	})
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]AggregatedGenreArtist, 0, len(items))
	for _, item := range items {
		record := store.Artists[item.slug]
		result = append(result, AggregatedGenreArtist{
			CanonicalSlug:       item.slug,
			Name:                record.Name,
			PlayCount:           item.count,
			SpotifyArtistID:     record.SpotifyArtistID,
			MusicBrainzArtistID: record.MusicBrainzArtistID,
			SpotifyURL:          record.SpotifyURL,
		})
	}
	return result
}

func topGenreReleases(store *genres.Store, counts map[string]int, limit int) []AggregatedGenreRelease {
	type bucket struct {
		slug  string
		count int
		name  string
	}
	items := make([]bucket, 0, len(counts))
	for slug, count := range counts {
		record, ok := store.Releases[slug]
		if !ok {
			continue
		}
		items = append(items, bucket{slug: slug, count: count, name: record.Name})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		if items[i].name != items[j].name {
			return items[i].name < items[j].name
		}
		return items[i].slug < items[j].slug
	})
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]AggregatedGenreRelease, 0, len(items))
	for _, item := range items {
		record := store.Releases[item.slug]
		result = append(result, AggregatedGenreRelease{
			CanonicalSlug:              item.slug,
			Name:                       record.Name,
			PlayCount:                  item.count,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			PrimaryArtistName:          record.PrimaryArtistName,
			SpotifyAlbumID:             record.SpotifyAlbumID,
			MusicBrainzReleaseID:       record.MusicBrainzReleaseID,
			MusicBrainzReleaseGroupID:  record.MusicBrainzReleaseGroupID,
		})
	}
	return result
}

func topGenreTracks(records map[string]derivedTrackRecord, counts map[string]int, limit int) []AggregatedGenreTrack {
	type bucket struct {
		slug  string
		count int
		name  string
	}
	items := make([]bucket, 0, len(counts))
	for slug, count := range counts {
		record, ok := records[slug]
		if !ok {
			continue
		}
		items = append(items, bucket{slug: slug, count: count, name: record.Name})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		if items[i].name != items[j].name {
			return items[i].name < items[j].name
		}
		return items[i].slug < items[j].slug
	})
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]AggregatedGenreTrack, 0, len(items))
	for _, item := range items {
		record := records[item.slug]
		result = append(result, AggregatedGenreTrack{
			CanonicalSlug:              item.slug,
			Name:                       record.Name,
			PlayCount:                  item.count,
			PrimaryArtistCanonicalSlug: record.PrimaryArtistSlug,
			PrimaryArtistName:          record.PrimaryArtistName,
			ReleaseCanonicalSlug:       record.ReleaseSlug,
			ReleaseName:                record.ReleaseName,
			SpotifyTrackID:             record.SpotifyTrackID,
			MusicBrainzTrackID:         record.MusicBrainzTrackID,
			SpotifyURL:                 record.SpotifyURL,
		})
	}
	return result
}

func genreSourceRefs(_ string, record AggregatedGenreRecord, includeTaxonomy bool) []AggregatedSourceReference {
	refs := []AggregatedSourceReference{}
	if includeTaxonomy {
		refs = append(refs, AggregatedSourceReference{
			Role:   "canonical_taxonomy",
			Source: "music-garden",
			Path:   filepath.ToSlash(filepath.Join("genre-taxonomy.json")),
			Note:   "Canonical genre slug, display title, aliases, parent, and notes come from the human-edited taxonomy.",
		})
	}
	if record.ListeningStats.PlayCount > 0 {
		refs = append(refs, AggregatedSourceReference{
			Role:      "listening_stats",
			Source:    "spotify",
			Path:      filepath.ToSlash(filepath.Join("plays")),
			UpdatedAt: record.ListeningStats.LastPlayedAt,
			Note:      "Listening stats and top artists/releases/tracks are derived from local canonicalized Spotify play history.",
		})
	}
	if record.WikipediaURL != "" || record.Summary != "" {
		ref := AggregatedSourceReference{
			Role:        "editorial_summary",
			Source:      "wikipedia",
			Path:        filepath.ToSlash(filepath.Join("normalized", "genres", "wikipedia--"+record.CanonicalSlug+".json")),
			URL:         record.WikipediaURL,
			RetrievedAt: "",
			UpdatedAt:   record.LastUpdated,
			Note:        "Editorial summary and page metadata come from Wikipedia normalization.",
		}
		if record.Attribution != nil {
			ref.URL = firstNonEmpty(record.Attribution.PageURL, ref.URL)
			ref.RetrievedAt = record.Attribution.RetrievedAt
		}
		refs = append(refs, ref)
	}
	hasMusicBrainz := false
	for _, artist := range record.TopArtists {
		if artist.MusicBrainzArtistID != "" {
			hasMusicBrainz = true
			break
		}
	}
	if !hasMusicBrainz {
		for _, release := range record.TopReleases {
			if release.MusicBrainzReleaseID != "" || release.MusicBrainzReleaseGroupID != "" {
				hasMusicBrainz = true
				break
			}
		}
	}
	if hasMusicBrainz {
		refs = append(refs, AggregatedSourceReference{
			Role:   "cross_source_enrichment",
			Source: "musicbrainz",
			Path:   filepath.ToSlash(filepath.Join("normalized")),
			Note:   "Top artists and releases carry MusicBrainz identifiers when enrichment is available.",
		})
	}
	if record.Image != nil {
		ref := AggregatedSourceReference{
			Role:   "image_metadata",
			Source: firstNonEmpty(record.Image.Source, "wikimedia-commons"),
			URL:    record.Image.FilePageURL,
			Path:   filepath.ToSlash(filepath.Join("normalized", "genres", "wikipedia--"+record.CanonicalSlug+".json")),
			Note:   "Image metadata is carried forward for later review before downstream use.",
		}
		if record.Attribution != nil && ref.URL == "" {
			ref.URL = record.Attribution.PageURL
		}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Role != refs[j].Role {
			return refs[i].Role < refs[j].Role
		}
		if refs[i].Source != refs[j].Source {
			return refs[i].Source < refs[j].Source
		}
		return refs[i].Path < refs[j].Path
	})
	return refs
}

func maxTimestamp(values ...string) string {
	best := ""
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if best == "" || value > best {
			best = value
		}
	}
	return best
}

func humanizeSlug(slug string) string {
	parts := strings.Split(strings.TrimSpace(slug), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
