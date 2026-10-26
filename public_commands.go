package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benstraw/music-garden/internal/plays"
	"github.com/benstraw/music-garden/internal/publicexport"
	"github.com/benstraw/music-garden/internal/reviews"
)

func runExport(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	contract := fs.String("contract", "public-v1", "export contract (public-v1)")
	out := fs.String("out", paths.publishedDir, "output directory")
	timezone := fs.String("timezone", "America/Los_Angeles", "publication timezone")
	sourceRevision := fs.String("source-revision", version, "source revision recorded in manifest")
	_ = fs.Parse(args)
	if *contract != "public-v1" {
		fmt.Fprintf(os.Stderr, "unsupported export contract %q\n", *contract)
		os.Exit(2)
	}

	catalog := loadCatalog(paths)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays:", err)
		os.Exit(1)
	}
	reviewStore, err := reviews.Load(paths.reviewsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load review decisions:", err)
		os.Exit(1)
	}
	outPath, err := filepath.Abs(*out)
	if err != nil {
		outPath = *out
	}
	manifest, err := publicexport.Export(publicexport.Options{
		OutputDir: outPath, SourceRevision: *sourceRevision, Timezone: *timezone,
		Catalog: catalog, Plays: allPlays, Reviews: reviewStore,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "export:", err)
		os.Exit(1)
	}
	saveCatalogOnly(paths, catalog)
	fmt.Printf("Exported public contract %s through %s to %s (%d artists, %d genres, %d releases, %d weeks)\n",
		manifest.ContractVersion, manifest.PublishedThrough, outPath,
		manifest.Counts.IndexedArtists, manifest.Counts.IndexedGenres,
		manifest.Counts.ListenedReleases, manifest.Counts.WeeklyPages)
}

func runReview(args []string, paths runtimePaths) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: music-garden review <list|show|approve|reject> --kind KIND [--slug SLUG]")
		os.Exit(2)
	}
	action := args[0]
	fs := flag.NewFlagSet("review "+action, flag.ExitOnError)
	kind := fs.String("kind", "", "artist, genre, release, or media")
	slug := fs.String("slug", "", "canonical slug or media review key")
	candidate := fs.String("candidate", "", "approved/rejected candidate identifier")
	reason := fs.String("reason", "", "review reason")
	force := fs.Bool("force-publish", false, "force page publication")
	suppress := fs.Bool("suppress", false, "suppress page publication")
	_ = fs.Parse(args[1:])
	if strings.TrimSpace(*kind) == "" {
		fmt.Fprintln(os.Stderr, "--kind is required")
		os.Exit(2)
	}
	store, err := reviews.Load(paths.reviewsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	switch action {
	case "list":
		for _, key := range store.Slugs(*kind) {
			decision, _ := store.Decision(*kind, key)
			fmt.Printf("%s\t%s\t%s\n", key, decision.Status, decision.Candidate)
		}
	case "show":
		decision, ok := store.Decision(*kind, *slug)
		if !ok {
			fmt.Fprintf(os.Stderr, "no %s review decision for %s\n", *kind, *slug)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(decision, "", "  ")
		fmt.Println(string(data))
	case "approve", "reject":
		if strings.TrimSpace(*slug) == "" {
			fmt.Fprintln(os.Stderr, "--slug is required")
			os.Exit(2)
		}
		status := "approved"
		if action == "reject" {
			status = "rejected"
		}
		decision := reviews.Decision{Status: status, Candidate: *candidate, Reason: *reason, ForcePublish: *force, Suppress: *suppress}
		if err := store.Set(*kind, *slug, decision); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := store.SaveKind(paths.reviewsDir, *kind); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%s %s review for %s\n", status, *kind, *slug)
	default:
		fmt.Fprintf(os.Stderr, "unknown review action %q\n", action)
		os.Exit(2)
	}
}
