package genres

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/benstraw/music-garden/internal/models"
)

const currentVersion = 8

const (
	WorkflowStateDraft       = "draft"
	WorkflowStatePublishable = "publishable"
)

// Store holds canonical artist/release metadata plus genre alias decisions.
type Store struct {
	Version             int                      `json:"version"`
	GenreAliases        map[string]string        `json:"genre_aliases,omitempty"`
	PendingGenreAliases []string                 `json:"pending_genre_aliases,omitempty"`
	GenreRecords        map[string]GenreRecord   `json:"genre_records,omitempty"`
	Artists             map[string]ArtistRecord  `json:"artists,omitempty"`
	ArtistSlugAliases   map[string]string        `json:"artist_slug_aliases,omitempty"`
	ArtistSourceIndex   map[string]string        `json:"artist_source_index,omitempty"`
	Releases            map[string]ReleaseRecord `json:"releases,omitempty"`
	ReleaseSourceIndex  map[string]string        `json:"release_source_index,omitempty"`
}

// ArtistRecord is the canonical metadata record for one artist.
type ArtistRecord struct {
	Slug                string                  `json:"slug"`
	Name                string                  `json:"name"`
	SortName            string                  `json:"sort_name,omitempty"`
	ArtistType          string                  `json:"artist_type,omitempty"`
	Country             string                  `json:"country,omitempty"`
	LifeSpan            *LifeSpan               `json:"life_span,omitempty"`
	Aliases             []string                `json:"aliases,omitempty"`
	SpotifyArtistID     string                  `json:"spotify_artist_id,omitempty"`
	MusicBrainzArtistID string                  `json:"musicbrainz_artist_id,omitempty"`
	WikidataID          string                  `json:"wikidata_id,omitempty"`
	SpotifyURL          string                  `json:"spotify_url,omitempty"`
	Genres              []string                `json:"genres,omitempty"`
	SourceGenres        []string                `json:"source_genres,omitempty"`
	Images              []models.ArtistImage    `json:"images,omitempty"`
	WikipediaTitle      string                  `json:"wikipedia_title,omitempty"`
	WikipediaURL        string                  `json:"wikipedia_url,omitempty"`
	Summary             string                  `json:"summary,omitempty"`
	Status              string                  `json:"status,omitempty"`
	Candidates          []string                `json:"candidates,omitempty"`
	Attribution         *GenreSourceAttribution `json:"attribution,omitempty"`
	Image               *GenreImageRecord       `json:"image,omitempty"`
	ImageCandidates     []GenreImageRecord      `json:"image_candidates,omitempty"`
	Relationships       []ArtistRelationship    `json:"relationships,omitempty"`
	LastUpdated         string                  `json:"last_updated,omitempty"`
}

type LifeSpan struct {
	Begin string `json:"begin,omitempty"`
	End   string `json:"end,omitempty"`
	Ended bool   `json:"ended,omitempty"`
}

type ArtistRelationship struct {
	Type       string `json:"type"`
	Direction  string `json:"direction,omitempty"`
	TargetSlug string `json:"target_slug,omitempty"`
	TargetName string `json:"target_name"`
	TargetMBID string `json:"target_musicbrainz_id,omitempty"`
	Begin      string `json:"begin,omitempty"`
	End        string `json:"end,omitempty"`
	SourceURL  string `json:"source_url,omitempty"`
}

// ReleaseRecord is the canonical metadata record for one release/album.
type ReleaseRecord struct {
	Slug                      string           `json:"slug"`
	Name                      string           `json:"name"`
	PrimaryArtistSlug         string           `json:"primary_artist_slug,omitempty"`
	PrimaryArtistName         string           `json:"primary_artist_name,omitempty"`
	SpotifyAlbumID            string           `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseGroupID string           `json:"musicbrainz_release_group_id,omitempty"`
	MusicBrainzReleaseID      string           `json:"musicbrainz_release_id,omitempty"`
	PrimaryType               string           `json:"primary_type,omitempty"`
	SecondaryTypes            []string         `json:"secondary_types,omitempty"`
	FirstReleaseDate          string           `json:"first_release_date,omitempty"`
	ArtistCredits             []PlayArtistRef  `json:"artist_credits,omitempty"`
	Editions                  []ReleaseEdition `json:"editions,omitempty"`
	LastUpdated               string           `json:"last_updated,omitempty"`
}

type PlayArtistRef struct {
	Slug      string `json:"slug,omitempty"`
	Name      string `json:"name"`
	MBID      string `json:"musicbrainz_id,omitempty"`
	SpotifyID string `json:"spotify_id,omitempty"`
}

type ReleaseEdition struct {
	SpotifyAlbumID       string               `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseID string               `json:"musicbrainz_release_id,omitempty"`
	Name                 string               `json:"name"`
	AlbumType            string               `json:"album_type,omitempty"`
	ReleaseDate          string               `json:"release_date,omitempty"`
	TotalTracks          int                  `json:"total_tracks"`
	Images               []models.ArtistImage `json:"images,omitempty"`
	Tracks               []ReleaseTrack       `json:"tracks,omitempty"`
}

type ReleaseTrack struct {
	DiscNumber     int    `json:"disc_number"`
	TrackNumber    int    `json:"track_number"`
	Name           string `json:"name"`
	SpotifyTrackID string `json:"spotify_track_id,omitempty"`
	MusicBrainzID  string `json:"musicbrainz_id,omitempty"`
	SpotifyURL     string `json:"spotify_url,omitempty"`
	DurationMS     int    `json:"duration_ms,omitempty"`
}

// GenreSourceAttribution preserves editorial/source provenance for a genre.
type GenreSourceAttribution struct {
	Source          string `json:"source,omitempty"`
	PageTitle       string `json:"page_title,omitempty"`
	PageURL         string `json:"page_url,omitempty"`
	License         string `json:"license,omitempty"`
	LicenseURL      string `json:"license_url,omitempty"`
	AttributionText string `json:"attribution_text,omitempty"`
	RetrievedAt     string `json:"retrieved_at,omitempty"`
}

// GenreImageRecord stores optional image metadata for a canonical genre page.
type GenreImageRecord struct {
	Source          string `json:"source,omitempty"`
	FileTitle       string `json:"file_title,omitempty"`
	FilePageURL     string `json:"file_page_url,omitempty"`
	ImageURL        string `json:"image_url,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	License         string `json:"license,omitempty"`
	LicenseURL      string `json:"license_url,omitempty"`
	Author          string `json:"author,omitempty"`
	AttributionText string `json:"attribution_text,omitempty"`
}

// GenreRecord stores canonical editorial metadata for one genre page.
type GenreRecord struct {
	Slug            string                  `json:"slug"`
	DisplayName     string                  `json:"display_name,omitempty"`
	ParentSlug      string                  `json:"parent_slug,omitempty"`
	Notes           string                  `json:"notes,omitempty"`
	WorkflowState   string                  `json:"workflow_state,omitempty"`
	WikipediaTitle  string                  `json:"wikipedia_title,omitempty"`
	WikipediaURL    string                  `json:"wikipedia_url,omitempty"`
	Summary         string                  `json:"summary,omitempty"`
	Status          string                  `json:"status,omitempty"`
	Candidates      []string                `json:"candidates,omitempty"`
	Attribution     *GenreSourceAttribution `json:"attribution,omitempty"`
	Image           *GenreImageRecord       `json:"image,omitempty"`
	ImageCandidates []GenreImageRecord      `json:"image_candidates,omitempty"`
	LastUpdated     string                  `json:"last_updated,omitempty"`
}

type legacyEntry struct {
	Name        string               `json:"name"`
	Genres      []string             `json:"genres"`
	Images      []models.ArtistImage `json:"images,omitempty"`
	LastUpdated string               `json:"last_updated"`
}

func NewStore() *Store {
	return &Store{
		Version:            currentVersion,
		GenreAliases:       map[string]string{},
		GenreRecords:       map[string]GenreRecord{},
		Artists:            map[string]ArtistRecord{},
		ArtistSlugAliases:  map[string]string{},
		ArtistSourceIndex:  map[string]string{},
		Releases:           map[string]ReleaseRecord{},
		ReleaseSourceIndex: map[string]string{},
	}
}

// Load reads the canonical metadata store from path. Legacy map-shaped caches
// are upgraded in memory to the canonical store shape on load.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewStore(), nil
	}
	if err != nil {
		return nil, err
	}
	return LoadBytes(data)
}

// LoadBytes reads the canonical metadata store from a JSON payload.
// Legacy map-shaped caches are upgraded in memory to the canonical store shape on load.
func LoadBytes(data []byte) (*Store, error) {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if _, ok := raw["version"]; ok || raw["artists"] != nil || raw["genre_aliases"] != nil {
		var store Store
		if err := json.Unmarshal(data, &store); err != nil {
			return nil, err
		}
		normalizeStore(&store)
		return &store, nil
	}

	legacy := map[string]legacyEntry{}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	store := NewStore()
	catalog := NewCatalog()
	catalog.Genres = store
	for spotifyID, entry := range legacy {
		rec := Update(catalog, spotifyID, entry.Name, "", entry.Genres, entry.Images)
		if entry.LastUpdated != "" {
			rec.LastUpdated = entry.LastUpdated
			store.Artists[rec.Slug] = rec
		}
	}
	return store, nil
}

// Save writes the canonical metadata store to path with 0644 permissions.
func Save(path string, store *Store) error {
	normalizeStore(store)
	sort.Strings(store.PendingGenreAliases)
	data, err := MarshalStore(store)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// MarshalStore serializes the genre-only canonical metadata store.
func MarshalStore(store *Store) ([]byte, error) {
	normalizeStore(store)
	persisted := persistedGenreStore{
		Version:             store.Version,
		GenreAliases:        store.GenreAliases,
		PendingGenreAliases: store.PendingGenreAliases,
		GenreRecords:        store.GenreRecords,
	}
	return json.MarshalIndent(persisted, "", "  ")
}

type persistedGenreStore struct {
	Version             int                    `json:"version"`
	GenreAliases        map[string]string      `json:"genre_aliases,omitempty"`
	PendingGenreAliases []string               `json:"pending_genre_aliases,omitempty"`
	GenreRecords        map[string]GenreRecord `json:"genre_records,omitempty"`
}

// UpsertGenreEditorial merges Wikipedia/Wikimedia editorial metadata into a canonical genre record.
func UpsertGenreEditorial(store *Store, record GenreRecord) GenreRecord {
	normalizeStore(store)
	if record.Slug == "" {
		return GenreRecord{}
	}
	existing := store.GenreRecords[record.Slug]
	existing.Slug = record.Slug
	existing.WikipediaTitle = firstNonEmpty(record.WikipediaTitle, existing.WikipediaTitle)
	existing.WikipediaURL = firstNonEmpty(record.WikipediaURL, existing.WikipediaURL)
	existing.Summary = firstNonEmpty(record.Summary, existing.Summary)
	existing.Status = firstNonEmpty(record.Status, existing.Status)
	if record.WorkflowState != "" {
		existing.WorkflowState = normalizeWorkflowState(record.WorkflowState)
	} else if existing.WorkflowState == "" {
		existing.WorkflowState = WorkflowStateDraft
	}
	if len(record.Candidates) > 0 {
		existing.Candidates = dedupeSortedStrings(record.Candidates)
	}
	if record.Attribution != nil {
		existing.Attribution = record.Attribution
	}
	if record.Image != nil {
		existing.Image = record.Image
	}
	if len(record.ImageCandidates) > 0 {
		existing.ImageCandidates = dedupeGenreImages(record.ImageCandidates)
	}
	existing.LastUpdated = today()
	store.GenreRecords[record.Slug] = existing
	return existing
}

// GenreEditorial returns the canonical editorial record for a genre slug, if any.
func GenreEditorial(store *Store, slug string) (GenreRecord, bool) {
	normalizeStore(store)
	record, ok := store.GenreRecords[slug]
	return record, ok
}

func GenreWorkflowState(record GenreRecord) string {
	return normalizeWorkflowState(record.WorkflowState)
}

func SetGenreWorkflowState(store *Store, slug, displayName, state string) GenreRecord {
	normalizeStore(store)
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return GenreRecord{}
	}
	record := store.GenreRecords[slug]
	record.Slug = slug
	if strings.TrimSpace(record.DisplayName) == "" {
		record.DisplayName = firstNonEmpty(displayName, humanizeGenreSlug(slug))
	}
	record.WorkflowState = normalizeWorkflowState(state)
	store.GenreRecords[slug] = record
	return record
}

// ResolvePlay fills garden-owned canonical IDs on a play while preserving
// source identifiers for provenance.
func ResolvePlay(source any, play models.Play) models.Play {
	catalog := catalogFor(source)
	if play.Source == "" {
		play.Source = "spotify"
	}
	if play.ArtistName != "" || play.ArtistID != "" {
		artist := ensureArtist(catalog, play.ArtistName, play.ArtistID, play.ArtistSpotifyURL)
		for _, extra := range play.AdditionalArtists {
			record := ensureArtist(catalog, extra.Name, extra.ID, extra.SpotifyURL)
			_ = record
		}
		release := ensureRelease(catalog, play.AlbumName, play.AlbumID, artist)
		release = mergeReleasePlay(release, play, artist)
		catalog.Releases.Releases[release.Slug] = release
		play.ArtistSlug = artist.Slug
		if play.ArtistMusicBrainzID == "" {
			play.ArtistMusicBrainzID = artist.MusicBrainzArtistID
		}
		if play.ArtistSpotifyURL == "" {
			play.ArtistSpotifyURL = artist.SpotifyURL
		}
		if play.ReleaseSlug == "" || play.ReleaseMusicBrainzID == "" || play.ReleaseGroupMusicBrainzID == "" {
			play.ReleaseSlug = release.Slug
			if play.ReleaseMusicBrainzID == "" {
				play.ReleaseMusicBrainzID = release.MusicBrainzReleaseID
			}
			if play.ReleaseGroupMusicBrainzID == "" {
				play.ReleaseGroupMusicBrainzID = release.MusicBrainzReleaseGroupID
			}
		}
		play.TrackSlug = derivedTrackSlug(play.ArtistSlug, play.TrackName, play.TrackID)
	}
	return play
}

func mergeReleasePlay(release ReleaseRecord, play models.Play, artist ArtistRecord) ReleaseRecord {
	release.PrimaryType = firstNonEmpty(release.PrimaryType, play.AlbumType)
	release.FirstReleaseDate = firstNonEmpty(release.FirstReleaseDate, play.AlbumReleaseDate)
	if len(release.ArtistCredits) == 0 {
		credits := play.AlbumArtists
		if len(credits) == 0 {
			credits = []models.PlayArtist{{ID: artist.SpotifyArtistID, Name: artist.Name}}
		}
		for _, credit := range credits {
			release.ArtistCredits = append(release.ArtistCredits, PlayArtistRef{
				Slug:      CanonicalArtistSlugForRecord(credit.Name),
				Name:      credit.Name,
				SpotifyID: credit.ID,
			})
		}
	}
	if play.AlbumID == "" {
		return release
	}
	editionIndex := -1
	for i := range release.Editions {
		if release.Editions[i].SpotifyAlbumID == play.AlbumID {
			editionIndex = i
			break
		}
	}
	if editionIndex < 0 {
		release.Editions = append(release.Editions, ReleaseEdition{
			SpotifyAlbumID: play.AlbumID,
			Name:           firstNonEmpty(play.AlbumName, release.Name),
			AlbumType:      play.AlbumType,
			ReleaseDate:    play.AlbumReleaseDate,
			TotalTracks:    play.AlbumTotalTracks,
			Images:         append([]models.ArtistImage(nil), play.AlbumImages...),
		})
		editionIndex = len(release.Editions) - 1
	}
	edition := release.Editions[editionIndex]
	edition.AlbumType = firstNonEmpty(edition.AlbumType, play.AlbumType)
	edition.ReleaseDate = firstNonEmpty(edition.ReleaseDate, play.AlbumReleaseDate)
	if edition.TotalTracks == 0 {
		edition.TotalTracks = play.AlbumTotalTracks
	}
	if len(edition.Images) == 0 {
		edition.Images = append([]models.ArtistImage(nil), play.AlbumImages...)
	}
	if play.TrackNumber > 0 {
		found := false
		for i := range edition.Tracks {
			if edition.Tracks[i].SpotifyTrackID == play.TrackID ||
				(edition.Tracks[i].DiscNumber == play.DiscNumber && edition.Tracks[i].TrackNumber == play.TrackNumber) {
				found = true
				break
			}
		}
		if !found {
			edition.Tracks = append(edition.Tracks, ReleaseTrack{
				DiscNumber:     play.DiscNumber,
				TrackNumber:    play.TrackNumber,
				Name:           play.TrackName,
				SpotifyTrackID: play.TrackID,
				MusicBrainzID:  play.TrackMusicBrainzID,
				SpotifyURL:     play.TrackSpotifyURL,
				DurationMS:     play.DurationMS,
			})
			sort.Slice(edition.Tracks, func(i, j int) bool {
				if edition.Tracks[i].DiscNumber != edition.Tracks[j].DiscNumber {
					return edition.Tracks[i].DiscNumber < edition.Tracks[j].DiscNumber
				}
				return edition.Tracks[i].TrackNumber < edition.Tracks[j].TrackNumber
			})
		}
	}
	release.Editions[editionIndex] = edition
	return release
}

func CanonicalArtistSlugForRecord(name string) string {
	return Slug(name)
}

func ReplaceArtistRecord(source any, record ArtistRecord) ArtistRecord {
	catalog := catalogFor(source)
	record.Aliases = dedupeSortedStrings(record.Aliases)
	catalog.Artists.Artists[record.Slug] = record
	return record
}

func ReplaceReleaseRecord(source any, record ReleaseRecord) ReleaseRecord {
	catalog := catalogFor(source)
	record.SecondaryTypes = dedupeSortedStrings(record.SecondaryTypes)
	catalog.Releases.Releases[record.Slug] = record
	return record
}

// MergeAlbumDetails replaces an observed-only edition tracklist with the
// ordered source edition returned by Spotify.
func MergeAlbumDetails(source any, album models.AlbumDetails) bool {
	catalog := catalogFor(source)
	slug, ok := catalog.Releases.ReleaseSourceIndex[sourceKey("spotify", album.ID)]
	if !ok {
		return false
	}
	release, ok := catalog.Releases.Releases[slug]
	if !ok {
		return false
	}
	editionIndex := -1
	for i := range release.Editions {
		if release.Editions[i].SpotifyAlbumID == album.ID {
			editionIndex = i
			break
		}
	}
	if editionIndex < 0 {
		release.Editions = append(release.Editions, ReleaseEdition{SpotifyAlbumID: album.ID})
		editionIndex = len(release.Editions) - 1
	}
	edition := release.Editions[editionIndex]
	edition.Name = firstNonEmpty(album.Name, edition.Name, release.Name)
	edition.AlbumType = firstNonEmpty(album.AlbumType, edition.AlbumType)
	edition.ReleaseDate = firstNonEmpty(album.ReleaseDate, edition.ReleaseDate)
	if album.TotalTracks > 0 {
		edition.TotalTracks = album.TotalTracks
	}
	if len(album.Images) > 0 {
		edition.Images = append([]models.ArtistImage(nil), album.Images...)
	}
	if len(album.Tracks) > 0 {
		edition.Tracks = make([]ReleaseTrack, 0, len(album.Tracks))
		for _, track := range album.Tracks {
			edition.Tracks = append(edition.Tracks, ReleaseTrack{
				DiscNumber: track.DiscNumber, TrackNumber: track.TrackNumber,
				Name: track.Name, SpotifyTrackID: track.ID,
				SpotifyURL: track.TrackSpotifyURL, DurationMS: track.DurationMS,
			})
		}
		sort.Slice(edition.Tracks, func(i, j int) bool {
			if edition.Tracks[i].DiscNumber != edition.Tracks[j].DiscNumber {
				return edition.Tracks[i].DiscNumber < edition.Tracks[j].DiscNumber
			}
			return edition.Tracks[i].TrackNumber < edition.Tracks[j].TrackNumber
		})
	}
	release.PrimaryType = firstNonEmpty(release.PrimaryType, album.AlbumType)
	release.FirstReleaseDate = firstNonEmpty(release.FirstReleaseDate, album.ReleaseDate)
	release.Editions[editionIndex] = edition
	release.LastUpdated = today()
	catalog.Releases.Releases[slug] = release
	return true
}

// ResolvePlays canonicalizes every play and reports whether any play changed.
func ResolvePlays(source any, plays []models.Play) ([]models.Play, bool) {
	catalog := catalogFor(source)
	resolved := make([]models.Play, len(plays))
	changed := false
	for i, play := range plays {
		r := ResolvePlay(catalog, play)
		if !reflect.DeepEqual(r, play) {
			changed = true
		}
		resolved[i] = r
	}
	return resolved, changed
}

// Update canonicalizes one Spotify artist payload into the store.
func Update(source any, spotifyID, name, spotifyURL string, sourceGenres []string, images []models.ArtistImage) ArtistRecord {
	catalog := catalogFor(source)
	record := ensureArtist(catalog, name, spotifyID, spotifyURL)
	record.Name = firstNonEmpty(name, record.Name)
	record.SpotifyArtistID = firstNonEmpty(spotifyID, record.SpotifyArtistID)
	record.SpotifyURL = firstNonEmpty(spotifyURL, record.SpotifyURL)
	record.SourceGenres = dedupeSortedStrings(append(record.SourceGenres, sourceGenres...))
	record.Genres = canonicalGenres(catalog.Genres, record.SourceGenres)
	if len(images) > 0 {
		record.Images = images
	}
	record.LastUpdated = today()
	catalog.Artists.Artists[record.Slug] = record
	return record
}

// UpsertArtistMetadata merges source metadata into a canonical artist record.
func UpsertArtistMetadata(source any, spotifyID, name, spotifyURL, musicBrainzID string, sourceGenres []string, images []models.ArtistImage) ArtistRecord {
	catalog := catalogFor(source)

	if musicBrainzID != "" {
		if slug, ok := catalog.Artists.ArtistSourceIndex[sourceKey("musicbrainz", musicBrainzID)]; ok {
			record := catalog.Artists.Artists[slug]
			if spotifyID != "" && record.SpotifyArtistID == "" {
				record.SpotifyArtistID = spotifyID
			}
			if spotifyURL != "" && record.SpotifyURL == "" {
				record.SpotifyURL = spotifyURL
			}
			if name != "" && record.Name == "" {
				record.Name = name
			}
			if len(sourceGenres) > 0 {
				record.SourceGenres = dedupeSortedStrings(append(record.SourceGenres, sourceGenres...))
				record.Genres = canonicalGenres(catalog.Genres, record.SourceGenres)
			}
			if len(images) > 0 {
				record.Images = images
			}
			record.MusicBrainzArtistID = musicBrainzID
			record.LastUpdated = today()
			catalog.Artists.Artists[slug] = record
			if spotifyID != "" {
				catalog.Artists.ArtistSourceIndex[sourceKey("spotify", spotifyID)] = slug
			}
			return record
		}
	}

	record := Update(catalog, spotifyID, name, spotifyURL, sourceGenres, images)
	if musicBrainzID != "" {
		record.MusicBrainzArtistID = musicBrainzID
		catalog.Artists.ArtistSourceIndex[sourceKey("musicbrainz", musicBrainzID)] = record.Slug
	}
	record.LastUpdated = today()
	catalog.Artists.Artists[record.Slug] = record
	return record
}

// UpsertArtistEditorial merges Wikipedia/Wikimedia editorial metadata into a canonical artist record.
func UpsertArtistEditorial(source any, slug string, record ArtistRecord) ArtistRecord {
	catalog := catalogFor(source)
	base, ok := catalog.Artists.Artists[slug]
	if !ok {
		base = ArtistRecord{Slug: slug, Name: record.Name}
	}
	base.WikipediaTitle = firstNonEmpty(record.WikipediaTitle, base.WikipediaTitle)
	base.WikipediaURL = firstNonEmpty(record.WikipediaURL, base.WikipediaURL)
	base.Summary = firstNonEmpty(record.Summary, base.Summary)
	base.Status = firstNonEmpty(record.Status, base.Status)
	if len(record.Candidates) > 0 {
		base.Candidates = dedupeSortedStrings(record.Candidates)
	}
	if record.Attribution != nil {
		base.Attribution = record.Attribution
	}
	if record.Image != nil {
		base.Image = record.Image
	}
	if len(record.ImageCandidates) > 0 {
		base.ImageCandidates = dedupeGenreImages(record.ImageCandidates)
	}
	base.LastUpdated = today()
	catalog.Artists.Artists[slug] = base
	return base
}

// UpsertReleaseMetadata merges source metadata into a canonical release record.
func UpsertReleaseMetadata(source any, artist ArtistRecord, spotifyAlbumID, name, musicBrainzReleaseGroupID, musicBrainzReleaseID string) ReleaseRecord {
	catalog := catalogFor(source)

	if musicBrainzReleaseGroupID != "" {
		if slug, ok := catalog.Releases.ReleaseSourceIndex[sourceKey("musicbrainz-release-group", musicBrainzReleaseGroupID)]; ok {
			record := catalog.Releases.Releases[slug]
			record.Name = firstNonEmpty(name, record.Name)
			record.PrimaryArtistSlug = firstNonEmpty(artist.Slug, record.PrimaryArtistSlug)
			record.PrimaryArtistName = firstNonEmpty(artist.Name, record.PrimaryArtistName)
			record.SpotifyAlbumID = firstNonEmpty(spotifyAlbumID, record.SpotifyAlbumID)
			record.MusicBrainzReleaseGroupID = firstNonEmpty(musicBrainzReleaseGroupID, record.MusicBrainzReleaseGroupID)
			record.MusicBrainzReleaseID = firstNonEmpty(musicBrainzReleaseID, record.MusicBrainzReleaseID)
			record.LastUpdated = today()
			catalog.Releases.Releases[slug] = record
			if spotifyAlbumID != "" {
				catalog.Releases.ReleaseSourceIndex[sourceKey("spotify", spotifyAlbumID)] = slug
			}
			if musicBrainzReleaseID != "" {
				catalog.Releases.ReleaseSourceIndex[sourceKey("musicbrainz-release", musicBrainzReleaseID)] = slug
			}
			return record
		}
	}

	if musicBrainzReleaseID != "" {
		if slug, ok := catalog.Releases.ReleaseSourceIndex[sourceKey("musicbrainz-release", musicBrainzReleaseID)]; ok {
			record := catalog.Releases.Releases[slug]
			record.Name = firstNonEmpty(name, record.Name)
			record.PrimaryArtistSlug = firstNonEmpty(artist.Slug, record.PrimaryArtistSlug)
			record.PrimaryArtistName = firstNonEmpty(artist.Name, record.PrimaryArtistName)
			record.SpotifyAlbumID = firstNonEmpty(spotifyAlbumID, record.SpotifyAlbumID)
			record.MusicBrainzReleaseGroupID = firstNonEmpty(musicBrainzReleaseGroupID, record.MusicBrainzReleaseGroupID)
			record.MusicBrainzReleaseID = firstNonEmpty(musicBrainzReleaseID, record.MusicBrainzReleaseID)
			record.LastUpdated = today()
			catalog.Releases.Releases[slug] = record
			if spotifyAlbumID != "" {
				catalog.Releases.ReleaseSourceIndex[sourceKey("spotify", spotifyAlbumID)] = slug
			}
			if musicBrainzReleaseGroupID != "" {
				catalog.Releases.ReleaseSourceIndex[sourceKey("musicbrainz-release-group", musicBrainzReleaseGroupID)] = slug
			}
			return record
		}
	}

	record := ensureRelease(catalog, name, spotifyAlbumID, artist)
	record.Name = firstNonEmpty(name, record.Name)
	record.PrimaryArtistSlug = firstNonEmpty(artist.Slug, record.PrimaryArtistSlug)
	record.PrimaryArtistName = firstNonEmpty(artist.Name, record.PrimaryArtistName)
	record.MusicBrainzReleaseGroupID = firstNonEmpty(musicBrainzReleaseGroupID, record.MusicBrainzReleaseGroupID)
	record.MusicBrainzReleaseID = firstNonEmpty(musicBrainzReleaseID, record.MusicBrainzReleaseID)
	record.LastUpdated = today()
	catalog.Releases.Releases[record.Slug] = record
	if musicBrainzReleaseGroupID != "" {
		catalog.Releases.ReleaseSourceIndex[sourceKey("musicbrainz-release-group", musicBrainzReleaseGroupID)] = record.Slug
	}
	if musicBrainzReleaseID != "" {
		catalog.Releases.ReleaseSourceIndex[sourceKey("musicbrainz-release", musicBrainzReleaseID)] = record.Slug
	}
	return record
}

// CanonicalizeTopArtist updates the store and returns a top artist value whose
// Genres field contains canonical genre slugs instead of raw source strings.
func CanonicalizeTopArtist(source any, artist models.TopArtist) models.TopArtist {
	catalog := catalogFor(source)
	record := Update(catalog, artist.ID, artist.Name, artist.SpotifyURL, artist.Genres, artist.Images)
	artist.ArtistSlug = record.Slug
	artist.MusicBrainzArtistID = record.MusicBrainzArtistID
	artist.SourceGenres = append([]string(nil), artist.Genres...)
	artist.Genres = append([]string(nil), record.Genres...)
	return artist
}

// UpdateImages updates only the images of an existing artist entry.
func UpdateImages(source any, spotifyID string, images []models.ArtistImage) {
	catalog := catalogFor(source)
	if spotifyID == "" {
		return
	}
	normalizeCatalog(catalog)
	slug, ok := catalog.Artists.ArtistSourceIndex[sourceKey("spotify", spotifyID)]
	if !ok {
		return
	}
	record := catalog.Artists.Artists[slug]
	record.Images = images
	record.LastUpdated = today()
	catalog.Artists.Artists[slug] = record
}

// MissingImagesArtistIDs returns Spotify artist IDs for records without images.
func MissingImagesArtistIDs(source any) []string {
	catalog := catalogFor(source)
	var ids []string
	for _, record := range catalog.Artists.Artists {
		if record.SpotifyArtistID != "" && len(record.Images) == 0 {
			ids = append(ids, record.SpotifyArtistID)
		}
	}
	sort.Strings(ids)
	return ids
}

// GenresForPlays returns canonical genres keyed by display artist name.
func GenresForPlays(source any, plays []models.Play) map[string][]string {
	catalog := catalogFor(source)
	result := map[string][]string{}
	for _, play := range plays {
		if play.ArtistName == "" {
			continue
		}
		if _, ok := result[play.ArtistName]; ok {
			continue
		}
		record, ok := artistForPlay(catalog, play)
		if ok && len(record.Genres) > 0 {
			result[play.ArtistName] = append([]string(nil), record.Genres...)
		}
	}
	return result
}

// ArtistsForPlays returns canonical artist records keyed by display artist name.
func ArtistsForPlays(source any, plays []models.Play) map[string]ArtistRecord {
	catalog := catalogFor(source)
	result := map[string]ArtistRecord{}
	for _, play := range plays {
		if play.ArtistName == "" {
			continue
		}
		if _, ok := result[play.ArtistName]; ok {
			continue
		}
		if record, ok := artistForPlay(catalog, play); ok {
			result[play.ArtistName] = record
		}
	}
	return result
}

// UncachedArtistIDs returns Spotify artist IDs not yet hydrated with canonical metadata.
func UncachedArtistIDs(source any, plays []models.Play) []string {
	catalog := catalogFor(source)
	seen := map[string]bool{}
	var ids []string
	for _, play := range plays {
		ids = append(ids, appendUncachedArtistID(catalog, seen, play.ArtistID)...)
		for _, extra := range play.AdditionalArtists {
			ids = append(ids, appendUncachedArtistID(catalog, seen, extra.ID)...)
		}
	}
	sort.Strings(ids)
	return ids
}

func appendUncachedArtistID(catalog *Catalog, seen map[string]bool, spotifyID string) []string {
	if spotifyID == "" || seen[spotifyID] {
		return nil
	}
	seen[spotifyID] = true
	slug, ok := catalog.Artists.ArtistSourceIndex[sourceKey("spotify", spotifyID)]
	if !ok {
		return []string{spotifyID}
	}
	record := catalog.Artists.Artists[slug]
	if len(record.Genres) == 0 && len(record.SourceGenres) == 0 {
		return []string{spotifyID}
	}
	return nil
}

// ArtistRecords returns all artist records sorted by name then slug.
func ArtistRecords(source any) []ArtistRecord {
	catalog := catalogFor(source)
	records := make([]ArtistRecord, 0, len(catalog.Artists.Artists))
	for _, record := range catalog.Artists.Artists {
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

// ReleaseRecords returns all release records sorted by name then slug.
func ReleaseRecords(source any) []ReleaseRecord {
	catalog := catalogFor(source)
	records := make([]ReleaseRecord, 0, len(catalog.Releases.Releases))
	for _, record := range catalog.Releases.Releases {
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

// CanonicalArtistSlug resolves a legacy or canonical artist slug to the
// canonical slug stored in the garden.
func CanonicalArtistSlug(source any, slug string) string {
	catalog := catalogFor(source)
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return ""
	}
	if _, ok := catalog.Artists.Artists[trimmed]; ok {
		return trimmed
	}
	normalized := Slug(trimmed)
	if _, ok := catalog.Artists.Artists[normalized]; ok {
		return normalized
	}
	if canonical, ok := catalog.Artists.ArtistSlugAliases[trimmed]; ok && canonical != "" {
		return canonical
	}
	if canonical, ok := catalog.Artists.ArtistSlugAliases[normalized]; ok && canonical != "" {
		return canonical
	}
	return normalized
}

// SetArtistSlugAlias records one legacy artist slug -> canonical slug mapping.
func SetArtistSlugAlias(source any, legacySlug, canonicalSlug string) {
	catalog := catalogFor(source)
	legacy := Slug(legacySlug)
	canonical := strings.TrimSpace(canonicalSlug)
	if legacy == "" || canonical == "" {
		return
	}
	if resolved := CanonicalArtistSlug(catalog, canonical); resolved != "" {
		canonical = resolved
	}
	if legacy == canonical {
		return
	}
	catalog.Artists.ArtistSlugAliases[legacy] = canonical
}

func artistForPlay(catalog *Catalog, play models.Play) (ArtistRecord, bool) {
	if play.ArtistSlug != "" {
		record, ok := catalog.Artists.Artists[CanonicalArtistSlug(catalog, play.ArtistSlug)]
		return record, ok
	}
	if play.ArtistID != "" {
		if slug, ok := catalog.Artists.ArtistSourceIndex[sourceKey("spotify", play.ArtistID)]; ok {
			record, ok := catalog.Artists.Artists[slug]
			return record, ok
		}
	}
	if play.ArtistName != "" {
		base := Slug(play.ArtistName)
		if record, ok := catalog.Artists.Artists[base]; ok && record.Name == play.ArtistName {
			return record, true
		}
	}
	return ArtistRecord{}, false
}

func ensureArtist(catalog *Catalog, name, spotifyID, spotifyURL string) ArtistRecord {
	if spotifyID != "" {
		if slug, ok := catalog.Artists.ArtistSourceIndex[sourceKey("spotify", spotifyID)]; ok {
			record := catalog.Artists.Artists[slug]
			if name != "" {
				record.Name = name
			}
			if spotifyURL != "" {
				record.SpotifyURL = spotifyURL
			}
			catalog.Artists.Artists[slug] = record
			return record
		}
	}

	base := Slug(name)
	if base == "" {
		base = "unknown-artist"
	}
	if canonical := CanonicalArtistSlug(catalog, base); canonical != "" {
		if existing, ok := catalog.Artists.Artists[canonical]; ok {
			if spotifyID != "" && existing.SpotifyArtistID == spotifyID {
				return existing
			}
			if spotifyID == "" && existing.Name == name {
				return existing
			}
		}
		base = canonical
	}
	slug := base
	for i := 2; ; i++ {
		existing, ok := catalog.Artists.Artists[slug]
		if !ok {
			break
		}
		if spotifyID != "" && existing.SpotifyArtistID == spotifyID {
			return existing
		}
		if spotifyID == "" && existing.Name == name {
			return existing
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}

	record := ArtistRecord{
		Slug:            slug,
		Name:            name,
		SpotifyArtistID: spotifyID,
		SpotifyURL:      spotifyURL,
		LastUpdated:     today(),
	}
	catalog.Artists.Artists[slug] = record
	if spotifyID != "" {
		catalog.Artists.ArtistSourceIndex[sourceKey("spotify", spotifyID)] = slug
	}
	return record
}

func ensureRelease(catalog *Catalog, name, spotifyAlbumID string, artist ArtistRecord) ReleaseRecord {
	if spotifyAlbumID != "" {
		if slug, ok := catalog.Releases.ReleaseSourceIndex[sourceKey("spotify", spotifyAlbumID)]; ok {
			record := catalog.Releases.Releases[slug]
			if name != "" {
				record.Name = name
			}
			catalog.Releases.Releases[slug] = record
			return record
		}
	}

	base := Slug(name)
	if base == "" {
		base = "unknown-release"
	}
	if artist.Slug != "" {
		base = artist.Slug + "--" + base
	}
	slug := base
	for i := 2; ; i++ {
		existing, ok := catalog.Releases.Releases[slug]
		if !ok {
			break
		}
		if spotifyAlbumID != "" && existing.SpotifyAlbumID == spotifyAlbumID {
			return existing
		}
		if spotifyAlbumID == "" && existing.Name == name && existing.PrimaryArtistSlug == artist.Slug {
			return existing
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}

	record := ReleaseRecord{
		Slug:              slug,
		Name:              name,
		PrimaryArtistSlug: artist.Slug,
		PrimaryArtistName: artist.Name,
		SpotifyAlbumID:    spotifyAlbumID,
		LastUpdated:       today(),
	}
	catalog.Releases.Releases[slug] = record
	if spotifyAlbumID != "" {
		catalog.Releases.ReleaseSourceIndex[sourceKey("spotify", spotifyAlbumID)] = slug
	}
	return record
}

func canonicalGenres(store *Store, sourceGenres []string) []string {
	normalizeStore(store)
	seen := map[string]bool{}
	var result []string
	for _, genre := range sourceGenres {
		canonical, ok := canonicalGenre(store, genre)
		if !ok || seen[canonical] {
			continue
		}
		seen[canonical] = true
		result = append(result, canonical)
	}
	sort.Strings(result)
	return result
}

// CanonicalGenre resolves one source genre string to a canonical slug.
func CanonicalGenre(store *Store, raw string) (string, bool) {
	return canonicalGenre(store, raw)
}

func canonicalGenre(store *Store, raw string) (string, bool) {
	key := normalizeLookup(raw)
	if key == "" {
		return "", false
	}
	canonical, ok := store.GenreAliases[key]
	if ok {
		return canonical, true
	}
	addPendingGenreAlias(store, raw)
	return "", false
}

func addPendingGenreAlias(store *Store, raw string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return
	}
	for _, existing := range store.PendingGenreAliases {
		if existing == trimmed {
			return
		}
	}
	store.PendingGenreAliases = append(store.PendingGenreAliases, trimmed)
}

func normalizeStore(store *Store) {
	if store.Version == 0 {
		store.Version = currentVersion
	}
	if store.Artists == nil {
		store.Artists = map[string]ArtistRecord{}
	}
	if store.ArtistSlugAliases == nil {
		store.ArtistSlugAliases = map[string]string{}
	}
	if store.ArtistSourceIndex == nil {
		store.ArtistSourceIndex = map[string]string{}
	}
	if store.Releases == nil {
		store.Releases = map[string]ReleaseRecord{}
	}
	if store.ReleaseSourceIndex == nil {
		store.ReleaseSourceIndex = map[string]string{}
	}
	if store.GenreAliases == nil {
		store.GenreAliases = map[string]string{}
	}
	if store.GenreRecords == nil {
		store.GenreRecords = map[string]GenreRecord{}
	}
	for slug, record := range store.GenreRecords {
		if strings.TrimSpace(record.Slug) == "" {
			record.Slug = slug
		}
		record.WorkflowState = normalizeWorkflowState(record.WorkflowState)
		store.GenreRecords[slug] = record
	}
	for alias, canonical := range store.ArtistSlugAliases {
		normalized := Slug(alias)
		delete(store.ArtistSlugAliases, alias)
		if normalized == "" {
			continue
		}
		resolved := strings.TrimSpace(canonical)
		if resolved == "" {
			continue
		}
		if canonicalSlug := CanonicalArtistSlug(store, resolved); canonicalSlug != "" {
			resolved = canonicalSlug
		}
		if normalized == resolved {
			continue
		}
		store.ArtistSlugAliases[normalized] = resolved
	}
}

func derivedTrackSlug(artistSlug, trackName, spotifyTrackID string) string {
	if artistSlug != "" && trackName != "" {
		return strings.TrimSpace(artistSlug) + "--" + Slug(trackName)
	}
	if trackName != "" {
		return Slug(trackName)
	}
	if spotifyTrackID != "" {
		return Slug("track-" + spotifyTrackID)
	}
	return ""
}

func normalizeWorkflowState(state string) string {
	switch strings.TrimSpace(strings.ToLower(state)) {
	case "", WorkflowStateDraft:
		return WorkflowStateDraft
	case WorkflowStatePublishable:
		return WorkflowStatePublishable
	default:
		return WorkflowStateDraft
	}
}

func dedupeGenreImages(images []GenreImageRecord) []GenreImageRecord {
	seen := map[string]bool{}
	result := make([]GenreImageRecord, 0, len(images))
	for _, image := range images {
		key := firstNonEmpty(image.FileTitle, image.FilePageURL, image.ImageURL, image.ThumbnailURL)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, image)
	}
	return result
}

func normalizeLookup(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case r == '&':
			if !lastSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString("and")
			lastSpace = false
		default:
			if !lastSpace && b.Len() > 0 {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// Slug converts a display string into a stable lowercase slug.
func Slug(s string) string {
	key := normalizeLookup(s)
	if key == "" {
		return ""
	}
	return strings.ReplaceAll(key, " ", "-")
}

func sourceKey(source, id string) string {
	return source + ":" + id
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
