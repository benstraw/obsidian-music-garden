package genreworkflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/genres"
)

const (
	RuleReadyBasic  = "ready-basic"
	RuleReadyLoose  = "ready-loose"
	RuleAnyPlay     = "any-play"
	RuleAnyPlayRich = "any-play-rich"
	RuleNoPlayRich  = "no-play-rich"
)

type GenreView struct {
	Slug             string
	DisplayName      string
	Pending          bool
	Mapped           bool
	WorkflowState    string
	Aliases          []string
	ParentSlug       string
	WikipediaTitle   string
	WikipediaURL     string
	Summary          string
	Status           string
	HasImage         bool
	ListeningStats   datalayer.GenreListeningStats
	TopArtistCount   int
	TopReleaseCount  int
	TopTrackCount    int
	AggregatedExists bool
}

func LoadViews(dataRoot string, store *genres.Store, taxonomy *genres.Taxonomy) ([]GenreView, error) {
	aggregatedDir := filepath.Join(dataRoot, "aggregated", "genres")
	records, err := datalayer.LoadAggregatedGenres(aggregatedDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	aggregatedBySlug := make(map[string]datalayer.AggregatedGenreRecord, len(records))
	for _, record := range records {
		aggregatedBySlug[record.CanonicalSlug] = record
	}

	taxonomySlugs := map[string]genres.GenreDefinition{}
	for _, definition := range taxonomy.Genres {
		slug := strings.TrimSpace(definition.Slug)
		if slug != "" {
			taxonomySlugs[slug] = definition
		}
	}

	aliasBuckets := map[string][]string{}
	for alias, slug := range store.GenreAliases {
		if strings.TrimSpace(slug) == "" {
			continue
		}
		aliasBuckets[slug] = append(aliasBuckets[slug], alias)
	}
	pendingBySlug := map[string]bool{}
	for _, pending := range store.PendingGenreAliases {
		slug := genres.Slug(pending)
		if slug != "" {
			pendingBySlug[slug] = true
		}
	}

	slugSet := map[string]bool{}
	for slug := range aggregatedBySlug {
		slugSet[slug] = true
	}
	for slug := range store.GenreRecords {
		slugSet[slug] = true
	}
	for slug := range taxonomySlugs {
		slugSet[slug] = true
	}
	for _, slug := range store.GenreAliases {
		if strings.TrimSpace(slug) != "" {
			slugSet[slug] = true
		}
	}
	for slug := range pendingBySlug {
		slugSet[slug] = true
	}

	slugs := make([]string, 0, len(slugSet))
	for slug := range slugSet {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	views := make([]GenreView, 0, len(slugs))
	for _, slug := range slugs {
		view := GenreView{
			Slug:          slug,
			DisplayName:   humanizeSlug(slug),
			Mapped:        taxonomySlugs[slug].Slug != "",
			Pending:       taxonomySlugs[slug].Slug == "" && pendingBySlug[slug],
			WorkflowState: genres.WorkflowStateDraft,
		}

		if definition, ok := taxonomySlugs[slug]; ok {
			view.DisplayName = firstNonEmpty(definition.DisplayName, view.DisplayName)
			view.ParentSlug = definition.ParentSlug
		}
		if record, ok := genres.GenreEditorial(store, slug); ok {
			view.DisplayName = firstNonEmpty(record.DisplayName, view.DisplayName)
			view.ParentSlug = firstNonEmpty(record.ParentSlug, view.ParentSlug)
			view.WorkflowState = genres.GenreWorkflowState(record)
			view.WikipediaTitle = record.WikipediaTitle
			view.WikipediaURL = record.WikipediaURL
			view.Summary = strings.TrimSpace(record.Summary)
			view.Status = record.Status
			view.HasImage = record.Image != nil
		}
		if aggregated, ok := aggregatedBySlug[slug]; ok {
			view.AggregatedExists = true
			view.DisplayName = firstNonEmpty(aggregated.DisplayName, aggregated.DisplayTitle, view.DisplayName)
			view.ParentSlug = firstNonEmpty(aggregated.ParentSlug, view.ParentSlug)
			view.Pending = aggregated.Pending
			view.WorkflowState = firstNonEmpty(strings.TrimSpace(aggregated.WorkflowState), view.WorkflowState)
			view.WikipediaTitle = firstNonEmpty(aggregated.WikipediaTitle, view.WikipediaTitle)
			view.WikipediaURL = firstNonEmpty(aggregated.WikipediaURL, view.WikipediaURL)
			view.Summary = firstNonEmpty(strings.TrimSpace(aggregated.Summary), view.Summary)
			view.Status = firstNonEmpty(aggregated.Status, view.Status)
			view.HasImage = view.HasImage || aggregated.Image != nil
			view.ListeningStats = aggregated.ListeningStats
			view.TopArtistCount = len(aggregated.TopArtists)
			view.TopReleaseCount = len(aggregated.TopReleases)
			view.TopTrackCount = len(aggregated.TopTracks)
			view.Aliases = append([]string(nil), aggregated.Aliases...)
		}
		if len(view.Aliases) == 0 {
			view.Aliases = append([]string(nil), aliasBuckets[slug]...)
			sort.Strings(view.Aliases)
		}
		views = append(views, view)
	}

	return views, nil
}

func QueueCandidates(views []GenreView, limit int) []GenreView {
	filtered := make([]GenreView, 0, len(views))
	for _, view := range views {
		if view.Pending || view.WorkflowState == genres.WorkflowStatePublishable {
			continue
		}
		filtered = append(filtered, view)
	}
	sortViews(filtered)
	if limit <= 0 || limit >= len(filtered) {
		return filtered
	}
	return filtered[:limit]
}

func EligibleForRule(view GenreView, rule string, minPlays int) (bool, error) {
	if view.Pending || !view.Mapped {
		return false, nil
	}
	switch strings.TrimSpace(strings.ToLower(rule)) {
	case RuleReadyBasic:
		return view.HasSummary() && view.ListeningStats.PlayCount >= minPlays, nil
	case RuleReadyLoose:
		return view.HasSummary() || view.ListeningStats.PlayCount >= minPlays, nil
	case RuleAnyPlay:
		return view.ListeningStats.PlayCount >= minPlays, nil
	case RuleAnyPlayRich:
		return view.ListeningStats.PlayCount >= minPlays && view.HasSummary() && view.HasImage, nil
	case RuleNoPlayRich:
		return view.HasSummary() && view.HasImage, nil
	default:
		return false, fmt.Errorf("unknown rule %q", rule)
	}
}

func PromoteCandidates(views []GenreView, rule string, minPlays, limit int) ([]GenreView, error) {
	filtered := make([]GenreView, 0, len(views))
	for _, view := range views {
		ok, err := EligibleForRule(view, rule, minPlays)
		if err != nil {
			return nil, err
		}
		if ok {
			filtered = append(filtered, view)
		}
	}
	sortViews(filtered)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func DefaultMinPlays(rule string) (int, error) {
	switch strings.TrimSpace(strings.ToLower(rule)) {
	case RuleReadyBasic:
		return 5, nil
	case RuleReadyLoose:
		return 10, nil
	case RuleAnyPlay:
		return 1, nil
	case RuleAnyPlayRich:
		return 1, nil
	case RuleNoPlayRich:
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown rule %q", rule)
	}
}

func SetWorkflowState(store *genres.Store, view GenreView, state string) genres.GenreRecord {
	return genres.SetGenreWorkflowState(store, view.Slug, view.DisplayName, state)
}

func FindView(views []GenreView, slug string) (GenreView, bool) {
	slug = genres.Slug(slug)
	for _, view := range views {
		if view.Slug == slug {
			return view, true
		}
	}
	return GenreView{}, false
}

func PageExists(outDir, slug string) bool {
	if strings.TrimSpace(outDir) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(outDir, slug+".md"))
	return err == nil
}

func (v GenreView) HasSummary() bool {
	return strings.TrimSpace(v.Summary) != ""
}

func (v GenreView) EnrichmentScore() int {
	score := 0
	if v.HasSummary() {
		score++
	}
	if v.HasImage {
		score++
	}
	if strings.TrimSpace(v.ParentSlug) != "" {
		score++
	}
	return score
}

func sortViews(views []GenreView) {
	sort.Slice(views, func(i, j int) bool {
		if views[i].ListeningStats.PlayCount != views[j].ListeningStats.PlayCount {
			return views[i].ListeningStats.PlayCount > views[j].ListeningStats.PlayCount
		}
		if views[i].EnrichmentScore() != views[j].EnrichmentScore() {
			return views[i].EnrichmentScore() > views[j].EnrichmentScore()
		}
		return views[i].Slug < views[j].Slug
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
