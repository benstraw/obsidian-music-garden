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

// AggregatedGenreRecord is the persisted canonical genre record.
type AggregatedGenreRecord struct {
	CanonicalSlug string   `json:"canonical_slug"`
	Aliases       []string `json:"aliases,omitempty"`
	Pending       bool     `json:"pending"`
}

// EnsureLayout creates the raw/normalized/aggregated directory layout.
func EnsureLayout(dataRoot string) error {
	dirs := []string{
		filepath.Join(dataRoot, "raw", "spotify", "recently-played"),
		filepath.Join(dataRoot, "raw", "spotify", "artists"),
		filepath.Join(dataRoot, "raw", "spotify", "top-artists"),
		filepath.Join(dataRoot, "raw", "musicbrainz"),
		filepath.Join(dataRoot, "raw", "wikipedia"),
		filepath.Join(dataRoot, "normalized", "artists"),
		filepath.Join(dataRoot, "normalized", "releases"),
		filepath.Join(dataRoot, "normalized", "genres"),
		filepath.Join(dataRoot, "aggregated", "artists"),
		filepath.Join(dataRoot, "aggregated", "releases"),
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
	if err := writeAggregatedArtists(filepath.Join(dataRoot, "aggregated", "artists"), store); err != nil {
		return err
	}
	if err := writeAggregatedReleases(filepath.Join(dataRoot, "aggregated", "releases"), store); err != nil {
		return err
	}
	if err := writeAggregatedGenres(filepath.Join(dataRoot, "aggregated", "genres"), store); err != nil {
		return err
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
	releases := make([]genres.ReleaseRecord, 0, len(store.Releases))
	for _, record := range store.Releases {
		releases = append(releases, record)
	}
	sort.Slice(releases, func(i, j int) bool {
		if releases[i].Name != releases[j].Name {
			return releases[i].Name < releases[j].Name
		}
		return releases[i].Slug < releases[j].Slug
	})
	for _, record := range releases {
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
