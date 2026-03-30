package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/genres"
)

type genrePageLink struct {
	Slug  string
	Title string
}

type genrePageData struct {
	Title              string
	CanonicalSlug      string
	Aliases            []string
	AliasesYAML        string
	ParentSlug         string
	ParentTitle        string
	Summary            string
	Status             string
	LastUpdated        string
	FirstPlayedAt      string
	LastPlayedAt       string
	PlayCount          int
	UniqueArtistCount  int
	UniqueReleaseCount int
	UniqueTrackCount   int
	TopArtists         []datalayer.AggregatedGenreArtist
	TopReleases        []datalayer.AggregatedGenreRelease
	TopTracks          []datalayer.AggregatedGenreTrack
	RelatedGenres      []genrePageLink
	SourceRefs         []datalayer.AggregatedSourceReference
	Notes              string
}

// RenderGenrePage builds deterministic markdown for one aggregated genre record.
func RenderGenrePage(record datalayer.AggregatedGenreRecord, all []datalayer.AggregatedGenreRecord, tmplPath string) (string, error) {
	data := genrePageData{
		Title:              firstNonEmpty(record.DisplayTitle, record.DisplayName, humanizeSlug(record.CanonicalSlug)),
		CanonicalSlug:      record.CanonicalSlug,
		Aliases:            record.Aliases,
		AliasesYAML:        yamlList(record.Aliases),
		ParentSlug:         record.ParentSlug,
		ParentTitle:        linkedGenreTitle(record.ParentSlug, all),
		Summary:            strings.TrimSpace(record.Summary),
		Status:             record.Status,
		LastUpdated:        record.LastUpdated,
		FirstPlayedAt:      record.ListeningStats.FirstPlayedAt,
		LastPlayedAt:       record.ListeningStats.LastPlayedAt,
		PlayCount:          record.ListeningStats.PlayCount,
		UniqueArtistCount:  record.ListeningStats.UniqueArtistCount,
		UniqueReleaseCount: record.ListeningStats.UniqueReleaseCount,
		UniqueTrackCount:   record.ListeningStats.UniqueTrackCount,
		TopArtists:         append([]datalayer.AggregatedGenreArtist(nil), record.TopArtists...),
		TopReleases:        append([]datalayer.AggregatedGenreRelease(nil), record.TopReleases...),
		TopTracks:          append([]datalayer.AggregatedGenreTrack(nil), record.TopTracks...),
		RelatedGenres:      relatedGenres(record, all),
		SourceRefs:         append([]datalayer.AggregatedSourceReference(nil), record.SourceRefs...),
		Notes:              strings.TrimSpace(record.Notes),
	}
	if data.Summary == "" {
		switch data.Status {
		case "matched":
			data.Summary = "Editorial summary not yet captured."
		case "ambiguous":
			data.Summary = "This genre still needs manual page selection because the source lookup was ambiguous."
		case "not_found":
			data.Summary = "No suitable encyclopedia summary has been matched for this genre yet."
		default:
			data.Summary = "This genre page is currently grounded in local listening data and canonical taxonomy only."
		}
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("parse genre template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render genre template: %w", err)
	}
	return buf.String(), nil
}

// WriteGenrePages renders aggregated genre records into outDir. It returns counts
// for updated, unchanged, and skipped files.
func WriteGenrePages(records []datalayer.AggregatedGenreRecord, outDir, tmplPath string, selected map[string]bool) (int, int, int, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return 0, 0, 0, err
	}
	updated := 0
	unchanged := 0
	skipped := 0
	for _, record := range records {
		if len(selected) > 0 && !selected[record.CanonicalSlug] {
			continue
		}
		// Skip non-publishable genres unless explicitly selected by slug.
		if (record.Pending || strings.TrimSpace(record.WorkflowState) != genres.WorkflowStatePublishable) && !selected[record.CanonicalSlug] {
			skipped++
			continue
		}
		content, err := RenderGenrePage(record, records, tmplPath)
		if err != nil {
			return updated, unchanged, skipped, err
		}
		path := filepath.Join(outDir, record.CanonicalSlug+".md")
		existing, err := os.ReadFile(path)
		if err == nil && string(existing) == content {
			unchanged++
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return updated, unchanged, skipped, err
		}
		updated++
	}
	return updated, unchanged, skipped, nil
}

func relatedGenres(record datalayer.AggregatedGenreRecord, all []datalayer.AggregatedGenreRecord) []genrePageLink {
	set := map[string]genrePageLink{}
	for _, candidate := range all {
		if candidate.CanonicalSlug == record.CanonicalSlug {
			continue
		}
		if record.ParentSlug != "" && candidate.CanonicalSlug == record.ParentSlug {
			set[candidate.CanonicalSlug] = genrePageLink{Slug: candidate.CanonicalSlug, Title: firstNonEmpty(candidate.DisplayTitle, candidate.DisplayName, humanizeSlug(candidate.CanonicalSlug))}
		}
		if candidate.ParentSlug == record.CanonicalSlug {
			set[candidate.CanonicalSlug] = genrePageLink{Slug: candidate.CanonicalSlug, Title: firstNonEmpty(candidate.DisplayTitle, candidate.DisplayName, humanizeSlug(candidate.CanonicalSlug))}
		}
		if record.ParentSlug != "" && candidate.ParentSlug == record.ParentSlug {
			set[candidate.CanonicalSlug] = genrePageLink{Slug: candidate.CanonicalSlug, Title: firstNonEmpty(candidate.DisplayTitle, candidate.DisplayName, humanizeSlug(candidate.CanonicalSlug))}
		}
	}
	links := make([]genrePageLink, 0, len(set))
	for _, link := range set {
		links = append(links, link)
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].Title != links[j].Title {
			return links[i].Title < links[j].Title
		}
		return links[i].Slug < links[j].Slug
	})
	return links
}

func linkedGenreTitle(slug string, all []datalayer.AggregatedGenreRecord) string {
	if strings.TrimSpace(slug) == "" {
		return ""
	}
	for _, record := range all {
		if record.CanonicalSlug == slug {
			return firstNonEmpty(record.DisplayTitle, record.DisplayName, humanizeSlug(record.CanonicalSlug))
		}
	}
	return humanizeSlug(slug)
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
