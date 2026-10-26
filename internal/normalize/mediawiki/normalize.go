package mediawikinormalize

import (
	"strings"
	"time"

	"github.com/benstraw/music-garden/internal/clients/mediawiki"
	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/genres"
)

// GenreSeed controls how one canonical genre slug should be enriched.
type GenreSeed struct {
	CanonicalSlug string `json:"canonical_slug"`
	SearchTerm    string `json:"search_term,omitempty"`
	PageTitle     string `json:"page_title,omitempty"`
}

// ArtistSeed controls how one canonical artist slug should be enriched.
type ArtistSeed struct {
	CanonicalSlug string `json:"canonical_slug"`
	Name          string `json:"name,omitempty"`
	PageTitle     string `json:"page_title,omitempty"`
	SearchTerm    string `json:"search_term,omitempty"`
}

// NormalizeMatchedGenre converts a Wikipedia match into normalized and canonical genre records.
func NormalizeMatchedGenre(seed GenreSeed, summary mediawiki.Summary, candidates []string, images []mediawiki.ImageInfo, fetchedAt time.Time) (datalayer.NormalizedGenreRecord, genres.GenreRecord) {
	record := genres.GenreRecord{
		Slug:           seed.CanonicalSlug,
		WikipediaTitle: firstNonEmpty(summary.Title, seed.PageTitle),
		WikipediaURL:   firstNonEmpty(summary.ContentURLs.Desktop.Page, summary.ContentURLs.Mobile.Page),
		Summary:        strings.TrimSpace(summary.Extract),
		Status:         "matched",
		Candidates:     append([]string(nil), candidates...),
		Attribution: &genres.GenreSourceAttribution{
			Source:      "wikipedia",
			PageTitle:   firstNonEmpty(summary.Title, seed.PageTitle),
			PageURL:     firstNonEmpty(summary.ContentURLs.Desktop.Page, summary.ContentURLs.Mobile.Page),
			RetrievedAt: fetchedAt.UTC().Format(time.RFC3339),
		},
	}
	record.ImageCandidates = toGenreImageCandidates(images)
	if len(record.ImageCandidates) > 0 {
		record.Image = &record.ImageCandidates[0]
	} else if summary.Thumbnail != nil || summary.Original != nil {
		record.Image = &genres.GenreImageRecord{
			Source:       "wikipedia",
			ImageURL:     originalURL(summary),
			ThumbnailURL: thumbnailURL(summary),
			Width:        summaryImageWidth(summary),
			Height:       summaryImageHeight(summary),
		}
	}
	record.LastUpdated = fetchedAt.UTC().Format("2006-01-02")
	normalized := datalayer.NormalizedGenreRecord{
		Source:             "wikipedia",
		SourceGenre:        firstNonEmpty(seed.SearchTerm, humanizeSlug(seed.CanonicalSlug)),
		CanonicalGenreSlug: seed.CanonicalSlug,
		Status:             "matched",
		PageTitle:          record.WikipediaTitle,
		Summary:            record.Summary,
		CanonicalURL:       record.WikipediaURL,
		Candidates:         append([]string(nil), candidates...),
		Attribution:        record.Attribution,
		Image:              record.Image,
		ImageCandidates:    append([]genres.GenreImageRecord(nil), record.ImageCandidates...),
	}
	return normalized, record
}

// NormalizeMatchedArtist converts a Wikipedia match into normalized and canonical artist records.
func NormalizeMatchedArtist(seed ArtistSeed, summary mediawiki.Summary, candidates []string, images []mediawiki.ImageInfo, fetchedAt time.Time) (datalayer.NormalizedArtistRecord, genres.ArtistRecord) {
	record := genres.ArtistRecord{
		Slug:           seed.CanonicalSlug,
		Name:           firstNonEmpty(seed.Name, summary.Title),
		WikipediaTitle: firstNonEmpty(summary.Title, seed.PageTitle),
		WikipediaURL:   firstNonEmpty(summary.ContentURLs.Desktop.Page, summary.ContentURLs.Mobile.Page),
		Summary:        strings.TrimSpace(summary.Extract),
		Status:         "matched",
		Candidates:     append([]string(nil), candidates...),
		Attribution: &genres.GenreSourceAttribution{
			Source:      "wikipedia",
			PageTitle:   firstNonEmpty(summary.Title, seed.PageTitle),
			PageURL:     firstNonEmpty(summary.ContentURLs.Desktop.Page, summary.ContentURLs.Mobile.Page),
			RetrievedAt: fetchedAt.UTC().Format(time.RFC3339),
		},
	}
	record.ImageCandidates = toGenreImageCandidates(images)
	if len(record.ImageCandidates) > 0 {
		record.Image = &record.ImageCandidates[0]
	} else if summary.Thumbnail != nil || summary.Original != nil {
		record.Image = &genres.GenreImageRecord{
			Source:       "wikipedia",
			ImageURL:     originalURL(summary),
			ThumbnailURL: thumbnailURL(summary),
			Width:        summaryImageWidth(summary),
			Height:       summaryImageHeight(summary),
		}
	}
	record.LastUpdated = fetchedAt.UTC().Format("2006-01-02")
	normalized := datalayer.NormalizedArtistRecord{
		Source:          "wikipedia",
		SourceArtistID:  seed.CanonicalSlug,
		Name:            record.Name,
		Status:          record.Status,
		PageTitle:       record.WikipediaTitle,
		Summary:         record.Summary,
		CanonicalURL:    record.WikipediaURL,
		Candidates:      append([]string(nil), candidates...),
		Attribution:     record.Attribution,
		Image:           record.Image,
		ImageCandidates: append([]genres.GenreImageRecord(nil), record.ImageCandidates...),
	}
	return normalized, record
}

// NormalizeUnresolvedArtist records an ambiguity or no-result outcome.
func NormalizeUnresolvedArtist(seed ArtistSeed, status string, candidates []string, fetchedAt time.Time) (datalayer.NormalizedArtistRecord, genres.ArtistRecord) {
	record := genres.ArtistRecord{
		Slug:        seed.CanonicalSlug,
		Name:        seed.Name,
		Status:      status,
		Candidates:  append([]string(nil), candidates...),
		LastUpdated: fetchedAt.UTC().Format("2006-01-02"),
	}
	normalized := datalayer.NormalizedArtistRecord{
		Source:         "wikipedia",
		SourceArtistID: seed.CanonicalSlug,
		Name:           seed.Name,
		Status:         status,
		Candidates:     append([]string(nil), candidates...),
	}
	return normalized, record
}

// NormalizeUnresolvedGenre records an ambiguity or no-result outcome.
func NormalizeUnresolvedGenre(seed GenreSeed, status string, candidates []string, fetchedAt time.Time) (datalayer.NormalizedGenreRecord, genres.GenreRecord) {
	record := genres.GenreRecord{
		Slug:        seed.CanonicalSlug,
		Status:      status,
		Candidates:  append([]string(nil), candidates...),
		LastUpdated: fetchedAt.UTC().Format("2006-01-02"),
	}
	normalized := datalayer.NormalizedGenreRecord{
		Source:             "wikipedia",
		SourceGenre:        firstNonEmpty(seed.SearchTerm, humanizeSlug(seed.CanonicalSlug)),
		CanonicalGenreSlug: seed.CanonicalSlug,
		Status:             status,
		Candidates:         append([]string(nil), candidates...),
	}
	return normalized, record
}

func humanizeSlug(slug string) string {
	return strings.ReplaceAll(strings.TrimSpace(slug), "-", " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func originalURL(summary mediawiki.Summary) string {
	if summary.Original != nil {
		return summary.Original.Source
	}
	return ""
}

func thumbnailURL(summary mediawiki.Summary) string {
	if summary.Thumbnail != nil {
		return summary.Thumbnail.Source
	}
	return ""
}

func summaryImageWidth(summary mediawiki.Summary) int {
	if summary.Original != nil && summary.Original.Width > 0 {
		return summary.Original.Width
	}
	if summary.Thumbnail != nil {
		return summary.Thumbnail.Width
	}
	return 0
}

func summaryImageHeight(summary mediawiki.Summary) int {
	if summary.Original != nil && summary.Original.Height > 0 {
		return summary.Original.Height
	}
	if summary.Thumbnail != nil {
		return summary.Thumbnail.Height
	}
	return 0
}

func toGenreImageCandidates(images []mediawiki.ImageInfo) []genres.GenreImageRecord {
	result := make([]genres.GenreImageRecord, 0, len(images))
	for _, image := range images {
		result = append(result, genres.GenreImageRecord{
			Source:          "wikimedia-commons",
			FileTitle:       image.FileTitle,
			FilePageURL:     image.FilePageURL,
			ImageURL:        image.ImageURL,
			ThumbnailURL:    image.ThumbnailURL,
			Width:           image.Width,
			Height:          image.Height,
			License:         image.License,
			LicenseURL:      image.LicenseURL,
			Author:          image.Author,
			AttributionText: image.AttributionText,
		})
	}
	return result
}
