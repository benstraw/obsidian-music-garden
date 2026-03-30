package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/enrich"
	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/genreworkflow"
	"github.com/benstraw/music-garden/internal/plays"
	"github.com/benstraw/music-garden/internal/render"
)

func runGenreReport(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("genre-report", flag.ExitOnError)
	taxonomyPath := fs.String("taxonomy", defaultGenreTaxonomyPath(paths), "canonical genre taxonomy path")
	_ = fs.Parse(args)

	taxonomy, err := genres.LoadTaxonomy(*taxonomyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre taxonomy error:", err)
		os.Exit(1)
	}

	catalog := loadCatalog(paths)
	report := genres.BuildGenreReport(taxonomy, catalog.Genres.PendingGenreAliases)
	fmt.Print(report.ReportString())
}

func runAggregateGenre(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("aggregate-genre", flag.ExitOnError)
	slug := fs.String("slug", "", "canonical genre slug")
	taxonomyPath := fs.String("taxonomy", defaultGenreTaxonomyPath(paths), "canonical genre taxonomy path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genre flags:", err)
		os.Exit(1)
	}
	genreSlug := genres.Slug(*slug)
	if genreSlug == "" {
		fmt.Fprintln(os.Stderr, "aggregate genre error: --slug is required")
		os.Exit(1)
	}

	catalog := loadCatalogWithTaxonomy(paths, *taxonomyPath)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genre load plays error:", err)
		os.Exit(1)
	}
	allPlays, _ = genres.ResolvePlays(catalog, allPlays)
	path, err := datalayer.RebuildAggregatedGenre(paths.dataRoot, catalog.Genres, allPlays, genreSlug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genre error:", err)
		os.Exit(1)
	}
	fmt.Printf("Aggregated genre %q -> %s\n", genreSlug, path)
}

func runAggregateGenres(paths runtimePaths) {
	catalog := loadCatalogWithTaxonomy(paths, defaultGenreTaxonomyPath(paths))
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genres load plays error:", err)
		os.Exit(1)
	}
	allPlays, _ = genres.ResolvePlays(catalog, allPlays)
	count, err := datalayer.RebuildAllAggregatedGenres(paths.dataRoot, catalog.Genres, allPlays)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genres error:", err)
		os.Exit(1)
	}
	fmt.Printf("Rebuilt %d aggregated genre record(s)\n", count)
}

func runSyncDataLayer(paths runtimePaths) {
	catalog := loadCatalogWithTaxonomy(paths, defaultGenreTaxonomyPath(paths))
	syncDataLayer(paths, catalog)
	fmt.Printf("Rebuilt normalized and aggregated data under %s\n", paths.dataRoot)
}

func runGenerateGenrePages(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("generate-genre-pages", flag.ExitOnError)
	outDir := fs.String("out-dir", defaultGenrePagesOutDir(paths), "output directory for generated genre markdown")
	slug := fs.String("slug", "", "optional canonical genre slug to generate")
	limit := fs.Int("limit", 0, "optional max number of pages to generate")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "generate genre pages flags:", err)
		os.Exit(1)
	}

	records, err := datalayer.LoadAggregatedGenres(filepath.Join(paths.dataRoot, "aggregated", "genres"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "load aggregated genres error:", err)
		os.Exit(1)
	}
	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "no aggregated genre records found")
		os.Exit(1)
	}

	selected := map[string]bool{}
	if strings.TrimSpace(*slug) != "" {
		selected[genres.Slug(*slug)] = true
	}
	if *limit > 0 && len(selected) == 0 {
		selected = map[string]bool{}
		for i, record := range records {
			if i >= *limit {
				break
			}
			selected[record.CanonicalSlug] = true
		}
	}

	tmplPath := filepath.Join(templatesDir(), "genre.md.tmpl")
	updated, unchanged, skipped, err := render.WriteGenrePages(records, *outDir, tmplPath, selected)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate genre pages error:", err)
		os.Exit(1)
	}
	fmt.Printf("Generated genre pages in %s (updated=%d unchanged=%d skipped=%d)\n", *outDir, updated, unchanged, skipped)
}

func runWikipediaBackfillGenres(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("wikipedia-backfill-genres", flag.ExitOnError)
	limit := fs.Int("limit", 0, "max genres to enrich")
	refresh := fs.Bool("refresh", false, "refresh genres that already have matched Wikipedia metadata")
	slug := fs.String("slug", "", "enrich a single canonical genre slug (overrides limit)")
	mappingPath := fs.String("mapping", enrich.DefaultMappingPath(paths.cwd, paths.dataRoot, "genre-page-mapping.json"), "genre mapping JSON path")
	_ = fs.Parse(args)

	client, err := getMediaWikiClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mediawiki client error:", err)
		os.Exit(1)
	}
	mapping, err := enrich.LoadGenrePageMapping(*mappingPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre mapping error:", err)
		os.Exit(1)
	}

	catalog := loadCatalogWithTaxonomy(paths, defaultGenreTaxonomyPath(paths))
	slugs := enrich.CanonicalGenreSlugsForBackfill(catalog.Genres)
	if strings.TrimSpace(*slug) != "" {
		slugs = []string{genres.Slug(*slug)}
	}
	processed, skipped, failed, attempted := 0, 0, 0, 0
	for _, slug := range slugs {
		if record, ok := genres.GenreEditorial(catalog.Genres, slug); ok && !*refresh && record.Status == "matched" && record.WikipediaURL != "" {
			skipped++
			continue
		}
		if *limit > 0 && attempted >= *limit {
			break
		}
		attempted++
		seed := mapping[slug]
		seed.CanonicalSlug = slug
		if seed.SearchTerm == "" {
			seed.SearchTerm = strings.ReplaceAll(slug, "-", " ")
		}
		status, _, err := enrich.WikipediaGenre(paths.dataRoot, client, catalog, seed, rawWikipediaWriter(paths), stderrWarn)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "warning: wikipedia genre %s: %v\n", slug, err)
			continue
		}
		processed++
		fmt.Printf("Wikipedia genre: %s (%s)\n", slug, status)
	}
	saveCatalog(paths, catalog)
	fmt.Printf("Wikipedia genre backfill complete: processed=%d skipped=%d failed=%d\n", processed, skipped, failed)
}

func runGenreReview(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("genre-review", flag.ExitOnError)
	slug := fs.String("slug", "", "canonical genre slug to inspect")
	queue := fs.Bool("queue", false, "show ranked draft review queue")
	limit := fs.Int("limit", 20, "max queue entries to print")
	taxonomyPath := fs.String("taxonomy", defaultGenreTaxonomyPath(paths), "canonical genre taxonomy path")
	outDir := fs.String("out-dir", defaultGenrePagesOutDir(paths), "directory to check for generated genre pages")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "genre review flags:", err)
		os.Exit(1)
	}

	_, views := loadGenreWorkflowViews(paths, *taxonomyPath)
	if *queue {
		queueViews := genreworkflow.QueueCandidates(views, *limit)
		fmt.Printf("Genre review queue (%d shown)\n", len(queueViews))
		for _, view := range queueViews {
			fmt.Printf("- %s  plays=%d enrichment=%d aliases=%d summary=%t image=%t parent=%t\n",
				view.Slug,
				view.ListeningStats.PlayCount,
				view.EnrichmentScore(),
				len(view.Aliases),
				view.HasSummary(),
				view.HasImage,
				strings.TrimSpace(view.ParentSlug) != "",
			)
		}
		return
	}

	genreSlug := genres.Slug(*slug)
	if genreSlug == "" {
		fmt.Fprintln(os.Stderr, "genre review error: use --slug or --queue")
		os.Exit(1)
	}
	view, ok := genreworkflow.FindView(views, genreSlug)
	if !ok {
		fmt.Fprintf(os.Stderr, "genre review error: unknown genre slug %q\n", genreSlug)
		os.Exit(1)
	}

	fmt.Printf("Slug: %s\n", view.Slug)
	fmt.Printf("Display name: %s\n", view.DisplayName)
	if view.Pending {
		fmt.Println("Taxonomy: pending")
	} else if view.Mapped {
		fmt.Println("Taxonomy: mapped")
	} else {
		fmt.Println("Taxonomy: unknown")
	}
	fmt.Printf("Workflow state: %s\n", view.WorkflowState)
	if len(view.Aliases) > 0 {
		fmt.Printf("Aliases: %s\n", strings.Join(view.Aliases, ", "))
	} else {
		fmt.Println("Aliases: none")
	}
	fmt.Printf("Wikipedia title: %s\n", firstNonEmpty(view.WikipediaTitle, "(none)"))
	fmt.Printf("Wikipedia URL: %s\n", firstNonEmpty(view.WikipediaURL, "(none)"))
	fmt.Printf("Summary present: %t\n", view.HasSummary())
	fmt.Printf("Image present: %t\n", view.HasImage)
	fmt.Printf("Listening: plays=%d unique_artists=%d unique_releases=%d unique_tracks=%d\n",
		view.ListeningStats.PlayCount,
		view.ListeningStats.UniqueArtistCount,
		view.ListeningStats.UniqueReleaseCount,
		view.ListeningStats.UniqueTrackCount,
	)
	fmt.Printf("Top entities: artists=%d releases=%d tracks=%d\n", view.TopArtistCount, view.TopReleaseCount, view.TopTrackCount)
	fmt.Printf("Aggregated record: %t\n", view.AggregatedExists)
	fmt.Printf("Generated page: %t (%s)\n", genreworkflow.PageExists(*outDir, view.Slug), filepath.Join(*outDir, view.Slug+".md"))
}

func runGenrePromote(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("genre-promote", flag.ExitOnError)
	rule := fs.String("rule", "", "promotion rule: ready-basic or ready-loose")
	dryRun := fs.Bool("dry-run", false, "report changes without writing")
	limit := fs.Int("limit", 0, "max promotions to apply")
	minPlays := fs.Int("min-plays", 0, "override minimum play count for the selected rule")
	slug := fs.String("slug", "", "canonical genre slug to promote or demote")
	state := fs.String("state", "", "manual workflow state: draft or publishable")
	taxonomyPath := fs.String("taxonomy", defaultGenreTaxonomyPath(paths), "canonical genre taxonomy path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "genre promote flags:", err)
		os.Exit(1)
	}

	catalog, views := loadGenreWorkflowViews(paths, *taxonomyPath)

	if strings.TrimSpace(*slug) != "" {
		genreSlug := genres.Slug(*slug)
		if genreSlug == "" {
			fmt.Fprintln(os.Stderr, "genre promote error: invalid --slug")
			os.Exit(1)
		}
		if strings.TrimSpace(*state) == "" {
			fmt.Fprintln(os.Stderr, "genre promote error: --state is required with --slug")
			os.Exit(1)
		}
		view, ok := genreworkflow.FindView(views, genreSlug)
		if !ok {
			view = genreworkflow.GenreView{Slug: genreSlug, DisplayName: humanizeGenreSlug(genreSlug), WorkflowState: genres.WorkflowStateDraft}
		}
		normalizedState := normalizeGenreState(*state)
		if normalizedState == "" {
			fmt.Fprintf(os.Stderr, "genre promote error: invalid state %q\n", *state)
			os.Exit(1)
		}
		if normalizedState == genres.WorkflowStatePublishable && view.Pending {
			fmt.Fprintf(os.Stderr, "warning: %s is taxonomy-pending; batch generation will still skip it until the taxonomy is fixed\n", view.Slug)
		}
		if *dryRun {
			fmt.Printf("Would set %s -> %s\n", view.Slug, normalizedState)
			return
		}
		genreworkflow.SetWorkflowState(catalog.Genres, view, normalizedState)
		saveCatalog(paths, catalog)
		fmt.Printf("Set %s -> %s\n", view.Slug, normalizedState)
		return
	}

	if strings.TrimSpace(*rule) == "" {
		fmt.Fprintln(os.Stderr, "genre promote error: use --rule or --slug")
		os.Exit(1)
	}
	defaultMinPlays, err := genreworkflow.DefaultMinPlays(*rule)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre promote error:", err)
		os.Exit(1)
	}
	effectiveMinPlays := defaultMinPlays
	if *minPlays > 0 {
		effectiveMinPlays = *minPlays
	}
	candidates, err := genreworkflow.PromoteCandidates(views, *rule, effectiveMinPlays, *limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre promote error:", err)
		os.Exit(1)
	}
	if len(candidates) == 0 {
		fmt.Println("No genres matched the promotion rule.")
		return
	}
	for _, view := range candidates {
		fmt.Printf("%s plays=%d summary=%t image=%t parent=%t\n",
			view.Slug,
			view.ListeningStats.PlayCount,
			view.HasSummary(),
			view.HasImage,
			strings.TrimSpace(view.ParentSlug) != "",
		)
		if !*dryRun {
			genreworkflow.SetWorkflowState(catalog.Genres, view, genres.WorkflowStatePublishable)
		}
	}
	if *dryRun {
		fmt.Printf("Dry run: %d genre(s) would be promoted\n", len(candidates))
		return
	}
	saveCatalog(paths, catalog)
	fmt.Printf("Promoted %d genre(s) to publishable\n", len(candidates))
}

type genreDoctorCounts struct {
	Total       int
	Pending     int
	Mapped      int
	Draft       int
	Publishable int
	Aggregated  int
	Generated   int
}

func doctorGenreCounts(paths runtimePaths) (genreDoctorCounts, error) {
	_, views := loadGenreWorkflowViews(paths, defaultGenreTaxonomyPath(paths))
	counts := genreDoctorCounts{}
	counts.Total = len(views)
	for _, view := range views {
		if view.Pending {
			counts.Pending++
		}
		if view.Mapped {
			counts.Mapped++
		}
		if view.WorkflowState == genres.WorkflowStatePublishable {
			counts.Publishable++
		} else {
			counts.Draft++
		}
		if view.AggregatedExists {
			counts.Aggregated++
		}
	}
	counts.Generated = countGeneratedGenrePages(defaultGenrePagesOutDir(paths))
	return counts, nil
}

func loadGenreWorkflowViews(paths runtimePaths, taxonomyPath string) (*genres.Catalog, []genreworkflow.GenreView) {
	catalog := loadCatalog(paths)
	taxonomy, err := genres.LoadTaxonomy(taxonomyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre taxonomy error:", err)
		os.Exit(1)
	}
	genres.ApplyTaxonomy(catalog.Genres, taxonomy)
	views, err := genreworkflow.LoadViews(paths.dataRoot, catalog.Genres, taxonomy)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre workflow load error:", err)
		os.Exit(1)
	}
	return catalog, views
}

func defaultGenrePagesOutDir(paths runtimePaths) string {
	return filepath.Join(paths.cwd, "content", "genres")
}

func countGeneratedGenrePages(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		count++
	}
	return count
}

func normalizeGenreState(state string) string {
	switch strings.TrimSpace(strings.ToLower(state)) {
	case genres.WorkflowStateDraft:
		return genres.WorkflowStateDraft
	case genres.WorkflowStatePublishable:
		return genres.WorkflowStatePublishable
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func humanizeGenreSlug(slug string) string {
	parts := strings.Split(strings.TrimSpace(slug), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
