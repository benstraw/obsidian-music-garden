package importlegacy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
)

type Options struct {
	SourceDir   string
	DataRoot    string
	DryRun      bool
	Verbose     bool
	AuditGenres bool
}

type Summary struct {
	ArtistsAdded            int
	ReleasesAdded           int
	ArtistSlugMappings      int
	GenreSlugMappings       int
	UnresolvedGenreLabels   int
	UnresolvedGenreSlugs    int
	CompatibilityFilesWrote bool
}

type LegacyArtistSlugMapping struct {
	LegacySlug      string `json:"legacy_slug"`
	CanonicalSlug   string `json:"canonical_slug,omitempty"`
	SpotifyArtistID string `json:"spotify_artist_id,omitempty"`
	Name            string `json:"name,omitempty"`
	Status          string `json:"status"`
}

type LegacyGenreSlugMapping struct {
	LegacySlug    string `json:"legacy_slug"`
	CanonicalSlug string `json:"canonical_slug,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	Status        string `json:"status"`
}

type topArtistsFile struct {
	Items []legacyArtist `json:"items"`
}

type legacyArtist struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Genres       []string             `json:"genres"`
	ExternalURLs map[string]string    `json:"external_urls"`
	Images       []models.ArtistImage `json:"images"`
}

type artistsFileRecord struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	SpotifyURL string   `json:"spotify_url"`
	Genres     []string `json:"genres"`
}

type legacySimpleArtist struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	ExternalURLs map[string]string `json:"external_urls"`
}

func Run(source any, opts Options) (Summary, error) {
	sourceDir := strings.TrimSpace(opts.SourceDir)
	if sourceDir == "" {
		return Summary{}, fmt.Errorf("empty source dir")
	}
	var catalog *genres.Catalog
	switch v := source.(type) {
	case *genres.Catalog:
		catalog = v
	case *genres.Store:
		catalog = genres.NewCatalog()
		catalog.Genres = v
		catalog.Artists.Artists = v.Artists
		catalog.Artists.ArtistSlugAliases = v.ArtistSlugAliases
		catalog.Artists.ArtistSourceIndex = v.ArtistSourceIndex
		catalog.Releases.Releases = v.Releases
		catalog.Releases.ReleaseSourceIndex = v.ReleaseSourceIndex
	default:
		return Summary{}, fmt.Errorf("unsupported legacy import store type")
	}
	store := catalog.Genres
	beforeArtists := len(store.Artists)
	beforeReleases := len(store.Releases)
	beforePending := len(store.PendingGenreAliases)

	if err := importTopArtists(catalog, filepath.Join(sourceDir, "topArtists.json"), opts.Verbose); err != nil {
		return Summary{}, err
	}
	if err := importTopArtists(catalog, filepath.Join(sourceDir, "snapshot-2024-06.json"), opts.Verbose); err != nil {
		return Summary{}, err
	}
	if err := importArtistsSupplement(catalog, filepath.Join(sourceDir, "artists.json"), opts.Verbose); err != nil {
		return Summary{}, err
	}
	artistMappings := buildLegacyArtistSlugMappings(catalog, filepath.Join(sourceDir, "artists.json"))
	applyLegacyArtistSlugMappings(catalog, artistMappings)
	genreMappings := buildLegacyGenreSlugMappings(store, sourceDir)

	summary := Summary{
		ArtistsAdded:          len(store.Artists) - beforeArtists,
		ReleasesAdded:         len(store.Releases) - beforeReleases,
		ArtistSlugMappings:    countResolvedArtistMappings(artistMappings),
		GenreSlugMappings:     countResolvedGenreMappings(genreMappings),
		UnresolvedGenreLabels: len(store.PendingGenreAliases) - beforePending,
		UnresolvedGenreSlugs:  countUnresolvedGenreMappings(genreMappings),
	}

	if opts.AuditGenres {
		return summary, nil
	}

	if opts.DryRun {
		return summary, nil
	}

	if err := writeJSON(filepath.Join(opts.DataRoot, "legacy-artist-slugs.json"), artistMappings); err != nil {
		return summary, err
	}
	if err := writeJSON(filepath.Join(opts.DataRoot, "legacy-genre-slugs.json"), genreMappings); err != nil {
		return summary, err
	}
	summary.CompatibilityFilesWrote = true
	return summary, nil
}

func importTopArtists(catalog *genres.Catalog, path string, verbose bool) error {
	var payload topArtistsFile
	ok, err := readJSON(path, &payload)
	if err != nil || !ok {
		return err
	}
	for _, artist := range payload.Items {
		if verbose {
			fmt.Printf("legacy top artist: %s\n", artist.Name)
		}
		genres.UpsertArtistMetadata(
			catalog,
			artist.ID,
			artist.Name,
			artist.ExternalURLs["spotify"],
			"",
			artist.Genres,
			artist.Images,
		)
	}
	return nil
}

func importArtistsSupplement(catalog *genres.Catalog, path string, verbose bool) error {
	var payload map[string]artistsFileRecord
	ok, err := readJSON(path, &payload)
	if err != nil || !ok {
		return err
	}
	keys := make([]string, 0, len(payload))
	for slug := range payload {
		keys = append(keys, slug)
	}
	sort.Strings(keys)
	for _, slug := range keys {
		record := payload[slug]
		if verbose {
			fmt.Printf("legacy artist supplement: %s\n", record.Name)
		}
		genres.UpsertArtistMetadata(
			catalog,
			record.ID,
			record.Name,
			record.SpotifyURL,
			"",
			record.Genres,
			nil,
		)
	}
	return nil
}

func legacyAllowedArtistIDs(store *genres.Store, sourceDir string) (map[string]bool, error) {
	result := map[string]bool{}
	for _, record := range store.Artists {
		if strings.TrimSpace(record.SpotifyArtistID) != "" {
			result[record.SpotifyArtistID] = true
		}
	}

	var topArtists topArtistsFile
	ok, err := readJSON(filepath.Join(sourceDir, "topArtists.json"), &topArtists)
	if err != nil {
		return nil, err
	}
	if ok {
		for _, artist := range topArtists.Items {
			if strings.TrimSpace(artist.ID) != "" {
				result[artist.ID] = true
			}
		}
	}

	var supplement map[string]artistsFileRecord
	ok, err = readJSON(filepath.Join(sourceDir, "artists.json"), &supplement)
	if err != nil {
		return nil, err
	}
	if ok {
		for _, record := range supplement {
			if strings.TrimSpace(record.ID) != "" {
				result[record.ID] = true
			}
		}
	}

	return result, nil
}

func buildLegacyArtistSlugMappings(catalog *genres.Catalog, path string) []LegacyArtistSlugMapping {
	store := catalog.Genres
	var payload map[string]artistsFileRecord
	ok, err := readJSON(path, &payload)
	if err != nil || !ok {
		return nil
	}
	keys := make([]string, 0, len(payload))
	for slug := range payload {
		keys = append(keys, slug)
	}
	sort.Strings(keys)
	result := make([]LegacyArtistSlugMapping, 0, len(keys))
	for _, legacySlug := range keys {
		record := payload[legacySlug]
		mapping := LegacyArtistSlugMapping{
			LegacySlug:      legacySlug,
			SpotifyArtistID: record.ID,
			Name:            record.Name,
			Status:          "unresolved",
		}
		if record.ID != "" {
			if canonicalSlug, ok := store.ArtistSourceIndex["spotify:"+record.ID]; ok {
				mapping.CanonicalSlug = canonicalSlug
				mapping.Status = "mapped"
			}
		}
		result = append(result, mapping)
	}
	return result
}

func applyLegacyArtistSlugMappings(catalog *genres.Catalog, mappings []LegacyArtistSlugMapping) {
	for _, mapping := range mappings {
		if mapping.Status != "mapped" || mapping.CanonicalSlug == "" {
			continue
		}
		genres.SetArtistSlugAlias(catalog, mapping.LegacySlug, mapping.CanonicalSlug)
	}
}

func buildLegacyGenreSlugMappings(store *genres.Store, sourceDir string) []LegacyGenreSlugMapping {
	websiteRoot := filepath.Dir(filepath.Dir(sourceDir))
	publicDir := filepath.Join(websiteRoot, "public", "musical-genres")
	entries, err := os.ReadDir(publicDir)
	if err != nil {
		return nil
	}
	known := knownGenreSlugs(store)
	result := make([]LegacyGenreSlugMapping, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		mapping := LegacyGenreSlugMapping{
			LegacySlug:  slug,
			DisplayName: humanizeSlug(slug),
			Status:      "unresolved",
		}
		if known[slug] {
			mapping.CanonicalSlug = slug
			mapping.Status = "mapped"
		}
		result = append(result, mapping)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LegacySlug < result[j].LegacySlug
	})
	return result
}

func knownGenreSlugs(store *genres.Store) map[string]bool {
	result := map[string]bool{}
	for _, slug := range store.GenreAliases {
		if strings.TrimSpace(slug) != "" {
			result[slug] = true
		}
	}
	for _, label := range store.PendingGenreAliases {
		if slug := genres.Slug(label); slug != "" {
			result[slug] = true
		}
	}
	for slug := range store.GenreRecords {
		if strings.TrimSpace(slug) != "" {
			result[slug] = true
		}
	}
	return result
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

func readJSON(path string, dest any) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("decode %s: %w", path, err)
	}
	return true, nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func countResolvedArtistMappings(mappings []LegacyArtistSlugMapping) int {
	count := 0
	for _, mapping := range mappings {
		if mapping.Status == "mapped" {
			count++
		}
	}
	return count
}

func countResolvedGenreMappings(mappings []LegacyGenreSlugMapping) int {
	count := 0
	for _, mapping := range mappings {
		if mapping.Status == "mapped" {
			count++
		}
	}
	return count
}

func countUnresolvedGenreMappings(mappings []LegacyGenreSlugMapping) int {
	count := 0
	for _, mapping := range mappings {
		if mapping.Status != "mapped" {
			count++
		}
	}
	return count
}
