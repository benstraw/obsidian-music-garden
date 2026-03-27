package genres

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/benstraw/music-garden/internal/models"
)

const currentVersion = 2

// Store holds canonical artist/release metadata plus genre alias decisions.
type Store struct {
	Version             int                      `json:"version"`
	GenreAliases        map[string]string        `json:"genre_aliases,omitempty"`
	PendingGenreAliases []string                 `json:"pending_genre_aliases,omitempty"`
	Artists             map[string]ArtistRecord  `json:"artists,omitempty"`
	ArtistSourceIndex   map[string]string        `json:"artist_source_index,omitempty"`
	Releases            map[string]ReleaseRecord `json:"releases,omitempty"`
	ReleaseSourceIndex  map[string]string        `json:"release_source_index,omitempty"`
}

// ArtistRecord is the canonical metadata record for one artist.
type ArtistRecord struct {
	Slug                string               `json:"slug"`
	Name                string               `json:"name"`
	SpotifyArtistID     string               `json:"spotify_artist_id,omitempty"`
	MusicBrainzArtistID string               `json:"musicbrainz_artist_id,omitempty"`
	SpotifyURL          string               `json:"spotify_url,omitempty"`
	Genres              []string             `json:"genres,omitempty"`
	SourceGenres        []string             `json:"source_genres,omitempty"`
	Images              []models.ArtistImage `json:"images,omitempty"`
	LastUpdated         string               `json:"last_updated,omitempty"`
}

// ReleaseRecord is the canonical metadata record for one release/album.
type ReleaseRecord struct {
	Slug                      string `json:"slug"`
	Name                      string `json:"name"`
	PrimaryArtistSlug         string `json:"primary_artist_slug,omitempty"`
	PrimaryArtistName         string `json:"primary_artist_name,omitempty"`
	SpotifyAlbumID            string `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseGroupID string `json:"musicbrainz_release_group_id,omitempty"`
	MusicBrainzReleaseID      string `json:"musicbrainz_release_id,omitempty"`
	LastUpdated               string `json:"last_updated,omitempty"`
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
		GenreAliases:       defaultGenreAliases(),
		Artists:            map[string]ArtistRecord{},
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
	for spotifyID, entry := range legacy {
		rec := Update(store, spotifyID, entry.Name, "", entry.Genres, entry.Images)
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
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ResolvePlay fills garden-owned canonical IDs on a play while preserving
// source identifiers for provenance.
func ResolvePlay(store *Store, play models.Play) models.Play {
	normalizeStore(store)
	if play.Source == "" {
		play.Source = "spotify"
	}
	if play.ArtistName != "" || play.ArtistID != "" {
		artist := ensureArtist(store, play.ArtistName, play.ArtistID, play.ArtistSpotifyURL)
		play.ArtistSlug = artist.Slug
		if play.ArtistMusicBrainzID == "" {
			play.ArtistMusicBrainzID = artist.MusicBrainzArtistID
		}
		if play.ArtistSpotifyURL == "" {
			play.ArtistSpotifyURL = artist.SpotifyURL
		}
		if play.ReleaseSlug == "" || play.ReleaseMusicBrainzID == "" || play.ReleaseGroupMusicBrainzID == "" {
			release := ensureRelease(store, play.AlbumName, play.AlbumID, artist)
			play.ReleaseSlug = release.Slug
			if play.ReleaseMusicBrainzID == "" {
				play.ReleaseMusicBrainzID = release.MusicBrainzReleaseID
			}
			if play.ReleaseGroupMusicBrainzID == "" {
				play.ReleaseGroupMusicBrainzID = release.MusicBrainzReleaseGroupID
			}
		}
	}
	return play
}

// ResolvePlays canonicalizes every play and reports whether any play changed.
func ResolvePlays(store *Store, plays []models.Play) ([]models.Play, bool) {
	resolved := make([]models.Play, len(plays))
	changed := false
	for i, play := range plays {
		r := ResolvePlay(store, play)
		if r != play {
			changed = true
		}
		resolved[i] = r
	}
	return resolved, changed
}

// Update canonicalizes one Spotify artist payload into the store.
func Update(store *Store, spotifyID, name, spotifyURL string, sourceGenres []string, images []models.ArtistImage) ArtistRecord {
	normalizeStore(store)
	record := ensureArtist(store, name, spotifyID, spotifyURL)
	record.Name = firstNonEmpty(name, record.Name)
	record.SpotifyArtistID = firstNonEmpty(spotifyID, record.SpotifyArtistID)
	record.SpotifyURL = firstNonEmpty(spotifyURL, record.SpotifyURL)
	record.SourceGenres = dedupeSortedStrings(sourceGenres)
	record.Genres = canonicalGenres(store, sourceGenres)
	if len(images) > 0 {
		record.Images = images
	}
	record.LastUpdated = today()
	store.Artists[record.Slug] = record
	return record
}

// CanonicalizeTopArtist updates the store and returns a top artist value whose
// Genres field contains canonical genre slugs instead of raw source strings.
func CanonicalizeTopArtist(store *Store, artist models.TopArtist) models.TopArtist {
	record := Update(store, artist.ID, artist.Name, artist.SpotifyURL, artist.Genres, artist.Images)
	artist.ArtistSlug = record.Slug
	artist.MusicBrainzArtistID = record.MusicBrainzArtistID
	artist.SourceGenres = append([]string(nil), artist.Genres...)
	artist.Genres = append([]string(nil), record.Genres...)
	return artist
}

// UpdateImages updates only the images of an existing artist entry.
func UpdateImages(store *Store, spotifyID string, images []models.ArtistImage) {
	if spotifyID == "" {
		return
	}
	normalizeStore(store)
	slug, ok := store.ArtistSourceIndex[sourceKey("spotify", spotifyID)]
	if !ok {
		return
	}
	record := store.Artists[slug]
	record.Images = images
	record.LastUpdated = today()
	store.Artists[slug] = record
}

// MissingImagesArtistIDs returns Spotify artist IDs for records without images.
func MissingImagesArtistIDs(store *Store) []string {
	normalizeStore(store)
	var ids []string
	for _, record := range store.Artists {
		if record.SpotifyArtistID != "" && len(record.Images) == 0 {
			ids = append(ids, record.SpotifyArtistID)
		}
	}
	sort.Strings(ids)
	return ids
}

// GenresForPlays returns canonical genres keyed by display artist name.
func GenresForPlays(store *Store, plays []models.Play) map[string][]string {
	result := map[string][]string{}
	for _, play := range plays {
		if play.ArtistName == "" {
			continue
		}
		if _, ok := result[play.ArtistName]; ok {
			continue
		}
		record, ok := artistForPlay(store, play)
		if ok && len(record.Genres) > 0 {
			result[play.ArtistName] = append([]string(nil), record.Genres...)
		}
	}
	return result
}

// ArtistsForPlays returns canonical artist records keyed by display artist name.
func ArtistsForPlays(store *Store, plays []models.Play) map[string]ArtistRecord {
	result := map[string]ArtistRecord{}
	for _, play := range plays {
		if play.ArtistName == "" {
			continue
		}
		if _, ok := result[play.ArtistName]; ok {
			continue
		}
		if record, ok := artistForPlay(store, play); ok {
			result[play.ArtistName] = record
		}
	}
	return result
}

// UncachedArtistIDs returns Spotify artist IDs not yet hydrated with canonical metadata.
func UncachedArtistIDs(store *Store, plays []models.Play) []string {
	normalizeStore(store)
	seen := map[string]bool{}
	var ids []string
	for _, play := range plays {
		if play.ArtistID == "" || seen[play.ArtistID] {
			continue
		}
		seen[play.ArtistID] = true
		slug, ok := store.ArtistSourceIndex[sourceKey("spotify", play.ArtistID)]
		if !ok {
			ids = append(ids, play.ArtistID)
			continue
		}
		record := store.Artists[slug]
		if len(record.Genres) == 0 && len(record.SourceGenres) == 0 {
			ids = append(ids, play.ArtistID)
		}
	}
	sort.Strings(ids)
	return ids
}

// ArtistRecords returns all artist records sorted by name then slug.
func ArtistRecords(store *Store) []ArtistRecord {
	normalizeStore(store)
	records := make([]ArtistRecord, 0, len(store.Artists))
	for _, record := range store.Artists {
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

func artistForPlay(store *Store, play models.Play) (ArtistRecord, bool) {
	normalizeStore(store)
	if play.ArtistSlug != "" {
		record, ok := store.Artists[play.ArtistSlug]
		return record, ok
	}
	if play.ArtistID != "" {
		if slug, ok := store.ArtistSourceIndex[sourceKey("spotify", play.ArtistID)]; ok {
			record, ok := store.Artists[slug]
			return record, ok
		}
	}
	if play.ArtistName != "" {
		base := Slug(play.ArtistName)
		if record, ok := store.Artists[base]; ok && record.Name == play.ArtistName {
			return record, true
		}
	}
	return ArtistRecord{}, false
}

func ensureArtist(store *Store, name, spotifyID, spotifyURL string) ArtistRecord {
	if spotifyID != "" {
		if slug, ok := store.ArtistSourceIndex[sourceKey("spotify", spotifyID)]; ok {
			record := store.Artists[slug]
			if name != "" {
				record.Name = name
			}
			if spotifyURL != "" {
				record.SpotifyURL = spotifyURL
			}
			store.Artists[slug] = record
			return record
		}
	}

	base := Slug(name)
	if base == "" {
		base = "unknown-artist"
	}
	slug := base
	for i := 2; ; i++ {
		existing, ok := store.Artists[slug]
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
	store.Artists[slug] = record
	if spotifyID != "" {
		store.ArtistSourceIndex[sourceKey("spotify", spotifyID)] = slug
	}
	return record
}

func ensureRelease(store *Store, name, spotifyAlbumID string, artist ArtistRecord) ReleaseRecord {
	if spotifyAlbumID != "" {
		if slug, ok := store.ReleaseSourceIndex[sourceKey("spotify", spotifyAlbumID)]; ok {
			record := store.Releases[slug]
			if name != "" {
				record.Name = name
			}
			store.Releases[slug] = record
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
		existing, ok := store.Releases[slug]
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
	store.Releases[slug] = record
	if spotifyAlbumID != "" {
		store.ReleaseSourceIndex[sourceKey("spotify", spotifyAlbumID)] = slug
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
	for key, value := range defaultGenreAliases() {
		if _, ok := store.GenreAliases[key]; !ok {
			store.GenreAliases[key] = value
		}
	}
}

func defaultGenreAliases() map[string]string {
	return map[string]string{
		"ambient":           "ambient",
		"alternative rock":  "alternative-rock",
		"art rock":          "art-rock",
		"dance pop":         "dance-pop",
		"dream pop":         "dream-pop",
		"electronic":        "electronic",
		"folk":              "folk",
		"folk rock":         "folk-rock",
		"hip hop":           "hip-hop",
		"hip-hop":           "hip-hop",
		"hiphop":            "hip-hop",
		"indie pop":         "indie-pop",
		"indie rock":        "indie-rock",
		"pop":               "pop",
		"rap":               "rap",
		"rock":              "rock",
		"singer songwriter": "singer-songwriter",
		"singer-songwriter": "singer-songwriter",
	}
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
