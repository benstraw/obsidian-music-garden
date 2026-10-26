package enrich

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	mwclient "github.com/benstraw/music-garden/internal/clients/mediawiki"
	mbclient "github.com/benstraw/music-garden/internal/clients/musicbrainz"
	"github.com/benstraw/music-garden/internal/genres"
	mwnormalize "github.com/benstraw/music-garden/internal/normalize/mediawiki"
	mbnormalize "github.com/benstraw/music-garden/internal/normalize/musicbrainz"
)

// RawWriter is a callback for persisting raw API response snapshots.
// kind identifies the response category (e.g. "artists", "artist-search"),
// stem is the file-name stem, and the remaining fields carry fetch metadata.
type RawWriter func(kind, stem, endpoint, requestURL string, fetchedAt time.Time, body []byte, fromCache bool)

// WarnFunc is a callback for reporting non-fatal warnings to the caller.
// The signature matches fmt.Sprintf so callers can wire it to any logging
// backend (stderr, structured logger, etc.).
type WarnFunc func(format string, args ...any)

func ensureCatalog(source any) *genres.Catalog {
	switch v := source.(type) {
	case *genres.Catalog:
		return v
	case *genres.Store:
		catalog := genres.NewCatalog()
		catalog.Genres = v
		catalog.Artists.Artists = v.Artists
		catalog.Artists.ArtistSlugAliases = v.ArtistSlugAliases
		catalog.Artists.ArtistSourceIndex = v.ArtistSourceIndex
		catalog.Releases.Releases = v.Releases
		catalog.Releases.ReleaseSourceIndex = v.ReleaseSourceIndex
		return catalog
	default:
		panic("unsupported catalog source")
	}
}

// --- MusicBrainz enrichment ---

// PickArtistMatch selects the best MusicBrainz artist from search results,
// preferring an exact name match. Falls back to the first result.
func PickArtistMatch(seedName string, matches []mbclient.SearchArtistResult) mbclient.SearchArtistResult {
	needle := strings.ToLower(strings.TrimSpace(seedName))
	for _, match := range matches {
		if strings.ToLower(strings.TrimSpace(match.Name)) == needle {
			return match
		}
	}
	return matches[0]
}

// PickReleaseGroupMatch selects the best MusicBrainz release-group from search
// results, preferring an exact artist+title match. Falls back to the first result.
func PickReleaseGroupMatch(seedArtist, seedRelease string, matches []mbclient.SearchReleaseGroupResult) mbclient.SearchReleaseGroupResult {
	artistNeedle := strings.ToLower(strings.TrimSpace(seedArtist))
	releaseNeedle := strings.ToLower(strings.TrimSpace(seedRelease))
	for _, match := range matches {
		if strings.ToLower(strings.TrimSpace(match.Title)) != releaseNeedle {
			continue
		}
		if strings.ToLower(strings.TrimSpace(mbclient.PrimaryArtistName(match.ArtistCredit))) == artistNeedle {
			return match
		}
	}
	return matches[0]
}

// MusicBrainzArtist enriches a single artist via MusicBrainz. It searches or
// looks up by MBID, writes normalized records, and returns the canonical
// ArtistRecord ready to merge into the store.
func MusicBrainzArtist(dataRoot string, mb *mbclient.Client, source any, seed mbnormalize.ArtistSeed, mbid string, writeRaw RawWriter, warn WarnFunc) (genres.ArtistRecord, error) {
	catalog := ensureCatalog(source)
	var artistResult mbclient.FetchResult[mbclient.Artist]
	var err error
	if strings.TrimSpace(mbid) != "" {
		artistResult, err = mb.GetArtistByID(strings.TrimSpace(mbid))
		if err != nil {
			return genres.ArtistRecord{}, err
		}
		writeRaw("artists", strings.TrimSpace(mbid), artistResult.Endpoint, artistResult.RequestURL, artistResult.FetchedAt, artistResult.Body, artistResult.FromCache)
	} else {
		return genres.ArtistRecord{}, fmt.Errorf("MusicBrainz name-search matches require review; provide an approved --mbid for %q", seed.Name)
	}

	_, record := mbnormalize.NormalizeArtist(catalog, seed, artistResult.Value)
	return record, nil
}

// MusicBrainzAlbum enriches a single release via MusicBrainz release-group
// search/lookup, writes normalized records, and returns the canonical
// ReleaseRecord ready to merge into the store.
func MusicBrainzAlbum(dataRoot string, mb *mbclient.Client, source any, seed mbnormalize.ReleaseSeed, mbid string, writeRaw RawWriter, warn WarnFunc) (genres.ReleaseRecord, error) {
	catalog := ensureCatalog(source)
	var groupResult mbclient.FetchResult[mbclient.ReleaseGroup]
	var err error
	if strings.TrimSpace(mbid) != "" {
		groupResult, err = mb.GetReleaseGroupByID(strings.TrimSpace(mbid))
		if err != nil {
			return genres.ReleaseRecord{}, err
		}
		writeRaw("release-groups", strings.TrimSpace(mbid), groupResult.Endpoint, groupResult.RequestURL, groupResult.FetchedAt, groupResult.Body, groupResult.FromCache)
	} else {
		return genres.ReleaseRecord{}, fmt.Errorf("MusicBrainz name-search matches require review; provide an approved --mbid for %q", seed.Name)
	}

	_, releaseRecord, _, _ := mbnormalize.NormalizeRelease(catalog, seed, groupResult.Value)
	return releaseRecord, nil
}

// --- Wikipedia enrichment ---

// WikipediaGenre enriches a single genre via Wikipedia search + summary.
// Returns (status, matchedTitle, error). Status is "matched", "ambiguous", or "not_found".
func WikipediaGenre(dataRoot string, client *mwclient.Client, source any, seed mwnormalize.GenreSeed, writeRaw RawWriter, warn WarnFunc) (string, string, error) {
	catalog := ensureCatalog(source)
	fetchedAt := time.Now().UTC()
	var pageTitle string
	var candidates []string

	if seed.PageTitle != "" {
		pageTitle = seed.PageTitle
		candidates = []string{seed.PageTitle}
	} else {
		search, err := client.SearchPages(seed.SearchTerm)
		if err != nil {
			return "", "", err
		}
		searchStem := genres.Slug(seed.CanonicalSlug)
		writeRaw("search", searchStem, search.Endpoint, search.RequestURL, search.FetchedAt, search.Body, search.FromCache)
		candidates = make([]string, 0, len(search.Value))
		for _, candidate := range search.Value {
			candidates = append(candidates, candidate.Title)
		}
		pageTitle, err = ChooseWikipediaPageTitle(seed, search.Value)
		if err != nil {
			if seed.PageTitle == "" && len(candidates) > 0 {
				if fallback, ferr := genreMusicFallbackSearch(client, seed, writeRaw); ferr == nil && fallback != "" {
					pageTitle = fallback
					err = nil
				}
			}
		}
		if err != nil {
			status := UnresolvedStatus(search.Value)
			_, record := mwnormalize.NormalizeUnresolvedGenre(seed, status, candidates, fetchedAt)
			genres.UpsertGenreEditorial(catalog.Genres, record)
			return status, "", nil
		}
	}

	summary, err := client.GetSummary(pageTitle)
	if err != nil {
		return "", "", err
	}
	summaryStem := genres.Slug(seed.CanonicalSlug)
	writeRaw("summaries", summaryStem, summary.Endpoint, summary.RequestURL, summary.FetchedAt, summary.Body, summary.FromCache)
	if summary.Value.Type == "disambiguation" {
		resolved := false
		if seed.PageTitle == "" {
			if fallback, ferr := genreMusicFallbackSearch(client, seed, writeRaw); ferr == nil && fallback != "" {
				if summary2, err2 := client.GetSummary(fallback); err2 == nil && summary2.Value.Type != "disambiguation" {
					pageTitle = fallback
					summary = summary2
					writeRaw("summaries", summaryStem, summary2.Endpoint, summary2.RequestURL, summary2.FetchedAt, summary2.Body, summary2.FromCache)
					resolved = true
				}
			}
		}
		if !resolved {
			_, record := mwnormalize.NormalizeUnresolvedGenre(seed, "ambiguous", candidates, fetchedAt)
			genres.UpsertGenreEditorial(catalog.Genres, record)
			return "ambiguous", "", nil
		}
	}

	imageInfos := CollectWikipediaImageCandidates(client, pageTitle, summaryStem, writeRaw)
	_, record := mwnormalize.NormalizeMatchedGenre(seed, summary.Value, candidates, imageInfos, fetchedAt)
	genres.UpsertGenreEditorial(catalog.Genres, record)
	return "matched", record.WikipediaTitle, nil
}

// WikipediaArtist enriches a single artist via Wikipedia search + summary.
// Returns (status, matchedTitle, error). Status is "matched", "ambiguous", or "not_found".
func WikipediaArtist(dataRoot string, client *mwclient.Client, source any, seed mwnormalize.ArtistSeed, writeRaw RawWriter, warn WarnFunc) (string, string, error) {
	catalog := ensureCatalog(source)
	fetchedAt := time.Now().UTC()
	var pageTitle string
	var candidates []string

	if seed.PageTitle != "" {
		pageTitle = seed.PageTitle
		candidates = []string{seed.PageTitle}
	} else {
		search, err := client.SearchPages(seed.SearchTerm)
		if err != nil {
			return "", "", err
		}
		searchStem := genres.Slug(seed.CanonicalSlug)
		writeRaw("search", "artist--"+searchStem, search.Endpoint, search.RequestURL, search.FetchedAt, search.Body, search.FromCache)
		candidates = make([]string, 0, len(search.Value))
		for _, candidate := range search.Value {
			candidates = append(candidates, candidate.Title)
		}
		pageTitle, err = ChooseWikipediaArtistPageTitle(seed, search.Value)
		if err != nil {
			status := UnresolvedStatus(search.Value)
			_, record := mwnormalize.NormalizeUnresolvedArtist(seed, status, candidates, fetchedAt)
			genres.UpsertArtistEditorial(catalog, seed.CanonicalSlug, record)
			return status, "", nil
		}
	}

	summary, err := client.GetSummary(pageTitle)
	if err != nil {
		return "", "", err
	}
	summaryStem := genres.Slug(seed.CanonicalSlug)
	writeRaw("summaries", "artist--"+summaryStem, summary.Endpoint, summary.RequestURL, summary.FetchedAt, summary.Body, summary.FromCache)
	if summary.Value.Type == "disambiguation" {
		_, record := mwnormalize.NormalizeUnresolvedArtist(seed, "ambiguous", candidates, fetchedAt)
		genres.UpsertArtistEditorial(catalog, seed.CanonicalSlug, record)
		return "ambiguous", "", nil
	}

	imageInfos := CollectWikipediaImageCandidates(client, pageTitle, "artist--"+summaryStem, writeRaw)
	_, record := mwnormalize.NormalizeMatchedArtist(seed, summary.Value, candidates, imageInfos, fetchedAt)
	genres.UpsertArtistEditorial(catalog, seed.CanonicalSlug, record)
	return "matched", record.WikipediaTitle, nil
}

// --- Wikipedia helpers ---

// ChooseWikipediaPageTitle picks the best Wikipedia page from search results
// for a genre seed.
func ChooseWikipediaPageTitle(seed mwnormalize.GenreSeed, candidates []mwclient.SearchResult) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("no results")
	}
	if seed.PageTitle != "" {
		for _, candidate := range candidates {
			if strings.EqualFold(candidate.Title, seed.PageTitle) {
				return candidate.Title, nil
			}
		}
		return "", fmt.Errorf("configured page title %q not found", seed.PageTitle)
	}
	if len(candidates) == 1 {
		return candidates[0].Title, nil
	}
	musicTerm := strings.TrimSpace(seed.SearchTerm) + " music"
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Title, musicTerm) {
			return candidate.Title, nil
		}
	}
	return "", fmt.Errorf("ambiguous search results")
}

func genreMusicFallbackSearch(client *mwclient.Client, seed mwnormalize.GenreSeed, writeRaw RawWriter) (string, error) {
	musicTerm := strings.TrimSpace(seed.SearchTerm) + " music"
	if strings.TrimSpace(seed.SearchTerm) == "" {
		return "", nil
	}
	search, err := client.SearchPages(musicTerm)
	if err != nil {
		return "", err
	}
	stem := genres.Slug(seed.CanonicalSlug) + "--music"
	writeRaw("search", stem, search.Endpoint, search.RequestURL, search.FetchedAt, search.Body, search.FromCache)
	for _, candidate := range search.Value {
		if strings.EqualFold(candidate.Title, musicTerm) {
			return candidate.Title, nil
		}
	}
	for _, candidate := range search.Value {
		if strings.EqualFold(candidate.Title, seed.SearchTerm) {
			return candidate.Title, nil
		}
	}
	return "", nil
}

// ChooseWikipediaArtistPageTitle picks the best Wikipedia page from search
// results for an artist seed.
func ChooseWikipediaArtistPageTitle(seed mwnormalize.ArtistSeed, candidates []mwclient.SearchResult) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("no results")
	}
	if seed.PageTitle != "" {
		for _, candidate := range candidates {
			if strings.EqualFold(candidate.Title, seed.PageTitle) {
				return candidate.Title, nil
			}
		}
		return "", fmt.Errorf("configured page title %q not found", seed.PageTitle)
	}
	if len(candidates) == 1 {
		return candidates[0].Title, nil
	}
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Title, seed.Name) || strings.EqualFold(candidate.Title, seed.SearchTerm) {
			return candidate.Title, nil
		}
	}
	return "", fmt.Errorf("ambiguous search results")
}

// UnresolvedStatus returns "not_found" or "ambiguous" based on search results.
func UnresolvedStatus(candidates []mwclient.SearchResult) string {
	if len(candidates) == 0 {
		return "not_found"
	}
	return "ambiguous"
}

// CollectWikipediaImageCandidates fetches editorial image candidates from a
// Wikipedia page's linked images, filtering out non-editorial files.
func CollectWikipediaImageCandidates(client *mwclient.Client, pageTitle, stem string, writeRaw RawWriter) []mwclient.ImageInfo {
	var imageInfos []mwclient.ImageInfo
	pageImages, err := client.GetPageImages(pageTitle)
	if err != nil {
		return nil
	}
	writeRaw("page-images", stem, pageImages.Endpoint, pageImages.RequestURL, pageImages.FetchedAt, pageImages.Body, pageImages.FromCache)
	for _, fileTitle := range pageImages.Value {
		if !strings.HasPrefix(fileTitle, "File:") || !IsLikelyGenreEditorialImage(fileTitle) {
			continue
		}
		info, infoErr := client.GetCommonsImageInfo(fileTitle)
		if infoErr != nil {
			continue
		}
		if !IsLikelyGenreEditorialImage(info.Value.FileTitle) {
			continue
		}
		writeRaw("commons-images", stem+"--"+genres.Slug(fileTitle), info.Endpoint, info.RequestURL, info.FetchedAt, info.Body, info.FromCache)
		imageInfos = append(imageInfos, info.Value)
		if len(imageInfos) >= 5 {
			break
		}
	}
	return imageInfos
}

// IsLikelyGenreEditorialImage filters out Wikipedia utility images.
func IsLikelyGenreEditorialImage(fileTitle string) bool {
	title := strings.ToLower(strings.TrimSpace(fileTitle))
	if !strings.HasPrefix(title, "file:") {
		return false
	}
	denylist := []string{
		"lock", "padlock", "icon", "logo", "symbol", "emblem",
		"wiktionary", "wikiquote", "commons-logo", "disambig",
		"question_book", "help", "portal-puzzle", "nuvola",
	}
	for _, blocked := range denylist {
		if strings.Contains(title, blocked) {
			return false
		}
	}
	return true
}

// --- Mapping loaders ---

// LoadGenrePageMapping reads a genre→Wikipedia page override mapping from disk.
// Returns an empty map if the file does not exist.
func LoadGenrePageMapping(path string) (map[string]mwnormalize.GenreSeed, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]mwnormalize.GenreSeed{}, nil
	}
	if err != nil {
		return nil, err
	}
	var mapping map[string]mwnormalize.GenreSeed
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}

// LoadArtistPageMapping reads an artist→Wikipedia page override mapping from disk.
// Returns an empty map if the file does not exist.
func LoadArtistPageMapping(path string) (map[string]mwnormalize.ArtistSeed, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]mwnormalize.ArtistSeed{}, nil
	}
	if err != nil {
		return nil, err
	}
	var mapping map[string]mwnormalize.ArtistSeed
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}

// --- Backfill helpers ---

// CanonicalGenreSlugsForBackfill returns a sorted, deduplicated list of all
// known canonical genre slugs from the store's alias maps and genre records.
func CanonicalGenreSlugsForBackfill(store *genres.Store) []string {
	set := map[string]bool{}
	for _, slug := range store.GenreAliases {
		if strings.TrimSpace(slug) != "" {
			set[slug] = true
		}
	}
	for _, label := range store.PendingGenreAliases {
		if slug := genres.Slug(label); slug != "" {
			set[slug] = true
		}
	}
	for slug := range store.GenreRecords {
		if strings.TrimSpace(slug) != "" {
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

// DefaultMappingPath resolves a data-file path by checking CWD first, then dataRoot.
func DefaultMappingPath(cwd, dataRoot, filename string) string {
	cwdPath := filepath.Join(cwd, "data", filename)
	if info, err := os.Stat(cwdPath); err == nil && !info.IsDir() {
		return cwdPath
	}
	return filepath.Join(dataRoot, filename)
}
