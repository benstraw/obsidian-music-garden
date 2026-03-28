package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/benstraw/music-garden/internal/auth"
	"github.com/benstraw/music-garden/internal/client"
	mwclient "github.com/benstraw/music-garden/internal/clients/mediawiki"
	mbclient "github.com/benstraw/music-garden/internal/clients/musicbrainz"
	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/fetch"
	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
	mwnormalize "github.com/benstraw/music-garden/internal/normalize/mediawiki"
	mbnormalize "github.com/benstraw/music-garden/internal/normalize/musicbrainz"
	"github.com/benstraw/music-garden/internal/plays"
	"github.com/benstraw/music-garden/internal/render"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
var version = "v0.9.0-dev"

type runtimePaths struct {
	cwd            string
	stateDir       string
	dotEnvPath     string
	tokensPath     string
	playsPath      string
	playsDir       string
	genresPath     string
	dataRoot       string
	playsOverride  bool
	genresOverride bool
	dotEnvFallback bool
	tokensFallback bool
	playsFallback  bool
	genresFallback bool
}

func main() {
	paths := resolveRuntimePaths()
	if err := loadDotEnv(paths.dotEnvPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", paths.dotEnvPath, err)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	emitFallbackWarnings(paths, cmd)

	switch cmd {
	case "auth":
		runAuth(paths)
	case "collect":
		runCollect(paths)
	case "repair-plays":
		runRepairPlays(paths)
	case "weekly":
		runWeekly(args, paths)
	case "daily":
		runDaily(args, paths)
	case "catch-up":
		runCatchUp(args, paths)
	case "persona":
		runPersona(args, paths)
	case "genre-backfill":
		runGenreBackfill(paths)
	case "image-backfill":
		runImageBackfill(paths)
	case "setlist":
		runSetlist(args)
	case "musicbrainz-enrich-artist":
		runMusicBrainzEnrichArtist(args, paths)
	case "musicbrainz-enrich-album":
		runMusicBrainzEnrichAlbum(args, paths)
	case "musicbrainz-backfill-artists":
		runMusicBrainzBackfillArtists(args, paths)
	case "musicbrainz-backfill-albums":
		runMusicBrainzBackfillAlbums(args, paths)
	case "wikipedia-enrich-genre":
		runWikipediaEnrichGenre(args, paths)
	case "wikipedia-enrich-artist":
		runWikipediaEnrichArtist(args, paths)
	case "wikipedia-backfill-genres":
		runWikipediaBackfillGenres(args, paths)
	case "wikipedia-backfill-artists":
		runWikipediaBackfillArtists(args, paths)
	case "genre-report":
		runGenreReport(args, paths)
	case "aggregate-genre":
		runAggregateGenre(args, paths)
	case "aggregate-genres":
		runAggregateGenres(paths)
	case "generate-genre-pages":
		runGenerateGenrePages(args, paths)
	case "doctor":
		os.Exit(runDoctor(paths))
	case "version", "--version":
		fmt.Println("music-garden", version)
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`music-garden %s — multi-source music garden → Obsidian markdown

Usage:
  music-garden auth                           Authenticate the current Spotify collector via OAuth
  music-garden collect                        Fetch last 50 Spotify plays, canonicalize, dedup, append to sharded history
  music-garden repair-plays                   Rewrite sharded play history with canonical IDs and richer persisted fields
  music-garden weekly [--date YYYY-MM-DD] [--out-dir DIR]     Generate weekly note for date's ISO week (default: current)
  music-garden daily [--date YYYY-MM-DD] [--out-dir DIR]      Generate daily note for date (default: today)
  music-garden catch-up [--weeks N] [--out-dir DIR]           Generate missing weekly + daily notes (default: 8 weeks back)
  music-garden persona [--out-dir DIR]                        Regenerate Music Taste context pack
  music-garden genre-backfill                 Fetch canonical artist genres for uncached artists in play history
  music-garden image-backfill                 Fetch artist images for canonical artist records that have none
  music-garden setlist <artist> [--date DATE] Look up setlist on setlist.fm (default: today)
  music-garden musicbrainz-enrich-artist      Enrich one canonical artist from a Spotify seed
  music-garden musicbrainz-enrich-album       Enrich one canonical release from a Spotify album seed
  music-garden musicbrainz-backfill-artists   Backfill MusicBrainz enrichment across canonical artists
  music-garden musicbrainz-backfill-albums    Backfill MusicBrainz enrichment across canonical releases
  music-garden wikipedia-enrich-genre         Enrich one canonical genre slug from Wikipedia/Wikimedia
  music-garden wikipedia-enrich-artist        Enrich one canonical artist slug from Wikipedia/Wikimedia
  music-garden wikipedia-backfill-genres      Backfill Wikipedia/Wikimedia enrichment across canonical genres
  music-garden wikipedia-backfill-artists     Backfill Wikipedia/Wikimedia enrichment across canonical artists
  music-garden genre-report                   Report canonical genre mappings, unknown labels, and collisions
  music-garden aggregate-genre                Rebuild one aggregated genre record from canonical data + local listening
  music-garden aggregate-genres               Rebuild all aggregated genre records
  music-garden generate-genre-pages          Render markdown genre pages from aggregated genre JSON
  music-garden doctor                         Print effective runtime config and diagnostics
  music-garden version                        Print version

Flags:
  --date   Date in YYYY-MM-DD format (default: today)
  --weeks  Number of weeks to check (default: 8)
`, version)
}

func resolveRuntimePaths() runtimePaths {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		cwd = "."
	}

	p := runtimePaths{
		cwd:        cwd,
		dotEnvPath: filepath.Join(cwd, ".env"),
		tokensPath: filepath.Join(cwd, "tokens.json"),
		playsPath:  filepath.Join(cwd, "data", "plays.json"),
		playsDir:   filepath.Join(cwd, "data", "plays"),
		genresPath: filepath.Join(cwd, "data", "genres.json"),
		dataRoot:   filepath.Join(cwd, "data"),
	}

	stateDir := strings.TrimSpace(os.Getenv("MUSIC_STATE_DIR"))
	if stateDir == "" {
		// Auto-discover the well-known installed state dir if it exists.
		home, err := os.UserHomeDir()
		if err == nil {
			candidate := filepath.Join(home, "Library", "Application Support", "music-garden", "state")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				stateDir = candidate
			}
		}
	}
	if stateDir == "" {
		if playsDir := strings.TrimSpace(os.Getenv("MUSIC_PLAYS_DIR")); playsDir != "" {
			absPlaysDir, err := filepath.Abs(playsDir)
			if err != nil {
				absPlaysDir = playsDir
			}
			p.playsDir = absPlaysDir
			p.playsPath = filepath.Join(filepath.Dir(absPlaysDir), "plays.json")
			p.playsOverride = true
		}
		if genresPath := strings.TrimSpace(os.Getenv("MUSIC_GENRES_PATH")); genresPath != "" {
			absGenresPath, err := filepath.Abs(genresPath)
			if err != nil {
				absGenresPath = genresPath
			}
			p.genresPath = absGenresPath
			p.genresOverride = true
		}
		return p
	}

	absState, err := filepath.Abs(stateDir)
	if err != nil {
		absState = stateDir
	}
	p.stateDir = absState
	p.dataRoot = filepath.Join(absState, "data")

	p.dotEnvPath, p.dotEnvFallback = chooseStatePath(absState, ".env", p.dotEnvPath)
	p.tokensPath, p.tokensFallback = chooseStatePath(absState, "tokens.json", p.tokensPath)
	if playsDir := strings.TrimSpace(os.Getenv("MUSIC_PLAYS_DIR")); playsDir != "" {
		absPlaysDir, err := filepath.Abs(playsDir)
		if err != nil {
			absPlaysDir = playsDir
		}
		p.playsDir = absPlaysDir
		p.playsPath = filepath.Join(filepath.Dir(absPlaysDir), "plays.json")
		p.playsOverride = true
		p.playsFallback = false
	} else {
		p.playsPath, p.playsFallback = chooseStatePath(absState, filepath.Join("data", "plays.json"), p.playsPath)
		// Resolve playsDir with its own fallback: prefer state dir, fall back to CWD.
		statePlayDir := filepath.Join(absState, "data", "plays")
		if info, err := os.Stat(statePlayDir); err == nil && info.IsDir() {
			p.playsDir = statePlayDir
		} else {
			cwdPlayDir := filepath.Join(cwd, "data", "plays")
			if info, err := os.Stat(cwdPlayDir); err == nil && info.IsDir() {
				p.playsDir = cwdPlayDir
				p.playsFallback = true
			} else {
				// Default to state dir so collect can create it there.
				p.playsDir = statePlayDir
			}
		}
	}
	if genresPath := strings.TrimSpace(os.Getenv("MUSIC_GENRES_PATH")); genresPath != "" {
		absGenresPath, err := filepath.Abs(genresPath)
		if err != nil {
			absGenresPath = genresPath
		}
		p.genresPath = absGenresPath
		p.genresOverride = true
		p.genresFallback = false
	} else {
		p.genresPath, p.genresFallback = chooseStatePath(absState, filepath.Join("data", "genres.json"), p.genresPath)
	}

	return p
}

func chooseStatePath(stateDir, relPath, fallbackPath string) (string, bool) {
	statePath := filepath.Join(stateDir, relPath)
	_, err := os.Stat(statePath)
	if err == nil {
		return statePath, false
	}
	if os.IsNotExist(err) {
		return fallbackPath, true
	}
	return statePath, false
}

func emitFallbackWarnings(paths runtimePaths, cmd string) {
	if paths.stateDir == "" {
		return
	}
	if cmd == "help" || cmd == "--help" || cmd == "-h" || cmd == "version" || cmd == "--version" {
		return
	}

	if paths.dotEnvFallback {
		fmt.Fprintf(os.Stderr, "warning: MUSIC_STATE_DIR is set but %s was not found; falling back to %s\n", filepath.Join(paths.stateDir, ".env"), paths.dotEnvPath)
	}

	tokensUsed := cmd == "auth" || cmd == "collect" || cmd == "persona" || cmd == "genre-backfill" || cmd == "image-backfill" || cmd == "doctor"
	if tokensUsed && paths.tokensFallback {
		fmt.Fprintf(os.Stderr, "warning: MUSIC_STATE_DIR is set but %s was not found; falling back to %s\n", filepath.Join(paths.stateDir, "tokens.json"), paths.tokensPath)
	}

	playsUsed := cmd == "collect" || cmd == "weekly" || cmd == "daily" || cmd == "catch-up" || cmd == "persona" || cmd == "genre-backfill" || cmd == "aggregate-genre" || cmd == "aggregate-genres" || cmd == "doctor"
	if playsUsed && paths.playsFallback {
		fmt.Fprintf(os.Stderr, "warning: MUSIC_STATE_DIR is set but %s was not found; falling back to %s\n", filepath.Join(paths.stateDir, "data", "plays"), paths.playsDir)
	}

	genresUsed := cmd == "collect" || cmd == "weekly" || cmd == "daily" || cmd == "catch-up" || cmd == "persona" || cmd == "genre-backfill" || cmd == "image-backfill" || cmd == "aggregate-genre" || cmd == "aggregate-genres"
	if genresUsed && paths.genresFallback {
		fmt.Fprintf(os.Stderr, "warning: MUSIC_STATE_DIR is set but %s was not found; falling back to %s\n", filepath.Join(paths.stateDir, "data", "genres.json"), paths.genresPath)
	}
}

// loadDotEnv reads a .env file and sets environment variables.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // .env is optional
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			os.Setenv(key, val)
		}
	}

	return scanner.Err()
}

// vaultPath returns the Obsidian vault path from the environment.
func vaultPath() string {
	v := os.Getenv("OBSIDIAN_VAULT_PATH")
	if v == "" {
		fmt.Fprintln(os.Stderr, "OBSIDIAN_VAULT_PATH not set")
		os.Exit(1)
	}
	return v
}

func noteOutputRoot(outDir string) string {
	if v := strings.TrimSpace(outDir); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			return abs
		}
		return v
	}
	return vaultPath()
}

// templatesDir returns the path to the templates directory.
func templatesDir() string {
	if td := os.Getenv("MUSIC_TEMPLATES_DIR"); td != "" {
		return td
	}
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "templates")
}

// getClient loads tokens (refreshing if needed) and returns an API client.
func getClient(paths runtimePaths) (*client.Client, error) {
	token, err := auth.RefreshIfNeeded(paths.tokensPath)
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w\nRun 'music-garden auth' to authenticate.", err)
	}
	return client.NewClient(token), nil
}

// parseDate parses a YYYY-MM-DD date string or returns today.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", s, err)
	}
	return t, nil
}

func envTrue(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ensurePlaysDir creates the parent directory for a runtime data file.
func ensurePlaysDir(playsPath string) error {
	return os.MkdirAll(filepath.Dir(playsPath), 0755)
}

func loadGenreStore(paths runtimePaths) *genres.Store {
	store, err := genres.Load(paths.genresPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: load metadata store: %v\n", err)
		store = genres.NewStore()
	}
	if taxonomy, err := genres.LoadTaxonomy(defaultGenreTaxonomyPath(paths)); err != nil {
		fmt.Fprintf(os.Stderr, "warning: load genre taxonomy: %v\n", err)
	} else {
		genres.ApplyTaxonomy(store, taxonomy)
	}
	return store
}

func saveGenreStore(paths runtimePaths, store *genres.Store) {
	if err := datalayer.EnsureLayout(paths.dataRoot); err != nil {
		fmt.Fprintf(os.Stderr, "warning: data layout error: %v\n", err)
	}
	if err := ensurePlaysDir(paths.genresPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: data dir error: %v\n", err)
		return
	}
	if err := genres.Save(paths.genresPath, store); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save metadata store: %v\n", err)
	}
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: load plays for aggregated sync: %v\n", err)
		allPlays = nil
	} else {
		allPlays, _ = genres.ResolvePlays(store, allPlays)
	}
	if err := datalayer.SyncAggregatedStore(paths.dataRoot, store, allPlays); err != nil {
		fmt.Fprintf(os.Stderr, "warning: sync aggregated data: %v\n", err)
	}
}

func migrateCanonicalPlays(paths runtimePaths, store *genres.Store) {
	changed, err := plays.MigrateCanonicalSharded(paths.playsDir, func(play models.Play) models.Play {
		return genres.ResolvePlay(store, play)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: canonical play migration: %v\n", err)
		return
	}
	if changed {
		saveGenreStore(paths, store)
	}
}

func canonicalizeTopArtists(store *genres.Store, artists []models.TopArtist) []models.TopArtist {
	canonical := make([]models.TopArtist, len(artists))
	for i, artist := range artists {
		canonical[i] = genres.CanonicalizeTopArtist(store, artist)
	}
	return canonical
}

func writeRawSpotifyRecentlyPlayed(paths runtimePaths, fetchedAt time.Time, body []byte) {
	if len(body) == 0 {
		return
	}
	if err := datalayer.EnsureLayout(paths.dataRoot); err != nil {
		fmt.Fprintf(os.Stderr, "warning: data layout error: %v\n", err)
		return
	}
	if _, err := datalayer.WriteRawSpotifyRecentlyPlayed(paths.dataRoot, fetchedAt, body); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write raw Spotify recently-played snapshot: %v\n", err)
	}
}

func writeRawSpotifyArtists(paths runtimePaths, fetchedAt time.Time, bodies [][]byte) {
	if len(bodies) == 0 {
		return
	}
	if err := datalayer.EnsureLayout(paths.dataRoot); err != nil {
		fmt.Fprintf(os.Stderr, "warning: data layout error: %v\n", err)
		return
	}
	for i, body := range bodies {
		if _, err := datalayer.WriteRawSpotifyArtists(paths.dataRoot, fetchedAt, i+1, body); err != nil {
			fmt.Fprintf(os.Stderr, "warning: write raw Spotify artists snapshot: %v\n", err)
			return
		}
	}
}

func writeRawSpotifyTopArtists(paths runtimePaths, fetchedAt time.Time, timeRange string, body []byte) {
	if len(body) == 0 {
		return
	}
	if err := datalayer.EnsureLayout(paths.dataRoot); err != nil {
		fmt.Fprintf(os.Stderr, "warning: data layout error: %v\n", err)
		return
	}
	if _, err := datalayer.WriteRawSpotifyTopArtists(paths.dataRoot, timeRange, fetchedAt, body); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write raw Spotify top-artists snapshot: %v\n", err)
	}
}

func musicBrainzUserAgent() string {
	if ua := strings.TrimSpace(os.Getenv("MUSICBRAINZ_USER_AGENT")); ua != "" {
		return ua
	}
	return fmt.Sprintf("obsidian-music-garden/%s ( https://github.com/benstraw/obsidian-music-garden )", version)
}

func getMusicBrainzClient(paths runtimePaths) (*mbclient.Client, error) {
	if err := datalayer.EnsureLayout(paths.dataRoot); err != nil {
		return nil, err
	}
	return mbclient.NewClient(mbclient.Config{
		UserAgent: musicBrainzUserAgent(),
		CacheDir:  filepath.Join(paths.dataRoot, "raw", "musicbrainz"),
		CacheTTL:  7 * 24 * time.Hour,
	})
}

func getMediaWikiClient(paths runtimePaths) (*mwclient.Client, error) {
	if err := datalayer.EnsureLayout(paths.dataRoot); err != nil {
		return nil, err
	}
	return mwclient.NewClient(mwclient.Config{
		UserAgent: fmt.Sprintf("obsidian-music-garden/%s ( https://github.com/benstraw/obsidian-music-garden )", version),
		CacheDir:  filepath.Join(paths.dataRoot, "raw", "wikipedia"),
		CacheTTL:  7 * 24 * time.Hour,
	})
}

func writeRawMusicBrainz(paths runtimePaths, kind, stem string, resultEndpoint, requestURL string, fetchedAt time.Time, body []byte, fromCache bool) {
	if len(body) == 0 {
		return
	}
	status := "fetched"
	if fromCache {
		status = "cached"
	}
	_, _, err := datalayer.WriteRawMusicBrainzResponse(paths.dataRoot, kind, stem, datalayer.RawFetchManifest{
		Source:      "musicbrainz",
		Endpoint:    resultEndpoint,
		RequestURL:  requestURL,
		CacheKey:    filepath.Join(kind, stem),
		FetchedAt:   fetchedAt.UTC().Format(time.RFC3339),
		Status:      status,
		ContentType: "application/json",
	}, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: write raw MusicBrainz %s snapshot: %v\n", kind, err)
	}
}

func writeRawWikipedia(paths runtimePaths, kind, stem string, endpoint, requestURL string, fetchedAt time.Time, body []byte, fromCache bool) {
	if len(body) == 0 {
		return
	}
	status := "fetched"
	if fromCache {
		status = "cached"
	}
	_, _, err := datalayer.WriteRawWikipediaResponse(paths.dataRoot, kind, stem, datalayer.RawFetchManifest{
		Source:      "wikipedia",
		Endpoint:    endpoint,
		RequestURL:  requestURL,
		CacheKey:    filepath.Join(kind, stem),
		FetchedAt:   fetchedAt.UTC().Format(time.RFC3339),
		Status:      status,
		ContentType: "application/json",
	}, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: write raw Wikipedia %s snapshot: %v\n", kind, err)
	}
}

// --- Subcommands ---

func runAuth(paths runtimePaths) {
	if err := auth.StartAuthFlow(paths.tokensPath); err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}
}

func runCollect(paths runtimePaths) {
	c, err := getClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)

	fmt.Println("Fetching recently played tracks...")
	fetchedAt := time.Now()
	incoming, rawRecentlyPlayed, err := fetch.GetRecentlyPlayedRaw(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch error:", err)
		os.Exit(1)
	}
	writeRawSpotifyRecentlyPlayed(paths, fetchedAt, rawRecentlyPlayed)

	// Migrate legacy plays.json to sharded structure on first use.
	if err := plays.MigrateToSharded(paths.playsPath, paths.playsDir); err != nil {
		fmt.Fprintln(os.Stderr, "migration error:", err)
		os.Exit(1)
	}
	migrateCanonicalPlays(paths, store)

	for i, play := range incoming {
		incoming[i] = genres.ResolvePlay(store, play)
	}

	newCount, err := plays.SaveSharded(paths.playsDir, incoming)
	if err != nil {
		fmt.Fprintln(os.Stderr, "save error:", err)
		os.Exit(1)
	}
	saveGenreStore(paths, store)

	fmt.Printf("Added %d new plays.\n", newCount)

	// Load all plays for metadata update and optional daily note.
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: load plays: %v\n", err)
		allPlays = incoming
	}

	// Update canonical artist metadata for any new artists.
	uncached := genres.UncachedArtistIDs(store, incoming)
	if len(uncached) > 0 {
		fmt.Printf("Fetching genres for %d new artist(s)...\n", len(uncached))
		artists, rawArtistBodies, err := fetch.GetArtistsBatchRaw(c, uncached)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: fetch artist genres: %v\n", err)
		} else {
			writeRawSpotifyArtists(paths, time.Now(), rawArtistBodies)
			for _, a := range artists {
				genres.CanonicalizeTopArtist(store, a)
			}
			saveGenreStore(paths, store)
		}
	}

	if envTrue("MUSIC_AUTO_DAILY_ON_COLLECT_SPOTIFY") {
		if os.Getenv("OBSIDIAN_VAULT_PATH") == "" {
			fmt.Fprintln(os.Stderr, "warning: MUSIC_AUTO_DAILY_ON_COLLECT_SPOTIFY is enabled but OBSIDIAN_VAULT_PATH is not set")
			return
		}
		artistMetadata := genres.ArtistsForPlays(store, allPlays)
		generateDailyNote(allPlays, time.Now(), true, artistMetadata, vaultPath())
	}
}

func runRepairPlays(paths runtimePaths) {
	store := loadGenreStore(paths)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays error:", err)
		os.Exit(1)
	}

	resolved, _ := genres.ResolvePlays(store, allPlays)

	resolvedByPlayedAt := make(map[string]models.Play, len(resolved))
	for _, play := range resolved {
		resolvedByPlayedAt[play.PlayedAt] = play
	}

	changed, err := plays.MigrateCanonicalSharded(paths.playsDir, func(play models.Play) models.Play {
		if resolved, ok := resolvedByPlayedAt[play.PlayedAt]; ok {
			return resolved
		}
		return genres.ResolvePlay(store, play)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "repair plays error:", err)
		os.Exit(1)
	}

	saveGenreStore(paths, store)
	if changed {
		fmt.Printf("Repaired sharded play history in %s\n", paths.playsDir)
		return
	}
	fmt.Printf("Sharded play history already canonical in %s\n", paths.playsDir)
}

func runWeekly(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("weekly", flag.ExitOnError)
	dateStr := fs.String("date", "", "any date within the target week (default: this week)")
	outDir := fs.String("out-dir", "", "output root override for safe sandbox generation")
	_ = fs.Parse(args)

	date, err := parseDate(*dateStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	generateWeeklyNote(date, paths, noteOutputRoot(*outDir))
}

// generateWeeklyNote filters local plays and writes the weekly summary note.
func generateWeeklyNote(date time.Time, paths runtimePaths, outputRoot string) {
	monday, _ := render.WeekBounds(date)
	weekStr := render.WeekStr(monday)

	outDir := filepath.Join(outputRoot, "music", "listening")
	outPath := filepath.Join(outDir, fmt.Sprintf("spotify-%s.md", weekStr))

	store := loadGenreStore(paths)
	migrateCanonicalPlays(paths, store)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays error:", err)
		os.Exit(1)
	}

	weekPlays := render.PlaysForWeek(allPlays, date)
	fmt.Printf("Plays for week %s: %d\n", weekStr, len(weekPlays))

	artistMetadata := genres.ArtistsForPlays(store, weekPlays)

	content, err := render.RenderWeekly(allPlays, date, outputRoot, artistMetadata)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render error:", err)
		os.Exit(1)
	}

	// Update artist stubs with canonical metadata
	for name, record := range artistMetadata {
		if err := render.UpdateArtistStub(record, outputRoot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: update artist stub for %s: %v\n", name, err)
		}
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir error:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
		os.Exit(1)
	}

	fmt.Println("Written:", outPath)
}

func generateDailyNote(allPlays []models.Play, date time.Time, overwrite bool, artistMetadata map[string]genres.ArtistRecord, outputRoot string) {
	d := date.Local()
	dayStr := d.Format("2006-01-02")
	outDir := filepath.Join(outputRoot, "music", "listening")
	outPath := filepath.Join(outDir, fmt.Sprintf("spotify-%s.md", dayStr))
	dayPlays := render.PlaysForDay(allPlays, date)
	if len(dayPlays) == 0 {
		return // no plays for this day
	}

	artistNames := map[string]bool{}
	for _, play := range dayPlays {
		artistNames[play.ArtistName] = true
	}
	artists := make([]string, 0, len(artistNames))
	for name := range artistNames {
		artists = append(artists, name)
	}
	sort.Strings(artists)
	for _, name := range artists {
		record, ok := artistMetadata[name]
		if !ok {
			record = genres.ArtistRecord{Name: name, Slug: genres.Slug(name)}
			for _, play := range dayPlays {
				if play.ArtistName == name {
					record.SpotifyArtistID = play.ArtistID
					record.SpotifyURL = play.ArtistSpotifyURL
					break
				}
			}
		}
		if err := render.EnsureArtistStub(record, dayStr, outputRoot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create artist stub for %s: %v\n", name, err)
		}
		if err := render.UpdateArtistStub(record, outputRoot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: update artist stub for %s: %v\n", name, err)
		}
	}

	// Skip existing daily notes unless overwrite is enabled.
	if _, err := os.Stat(outPath); err == nil && !overwrite {
		fmt.Printf("  Skipping %s (already exists)\n", dayStr)
		return
	}

	content, err := render.RenderDaily(allPlays, date, outputRoot, artistMetadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render error for %s: %v\n", dayStr, err)
		return
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir error: %v\n", err)
		return
	}
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write error for %s: %v\n", dayStr, err)
		return
	}
	if overwrite {
		fmt.Println("Updated:", outPath)
	} else {
		fmt.Println("Written:", outPath)
	}
}

func runDaily(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("daily", flag.ExitOnError)
	dateStr := fs.String("date", "", "date in YYYY-MM-DD format (default: today)")
	outDir := fs.String("out-dir", "", "output root override for safe sandbox generation")
	_ = fs.Parse(args)

	date, err := parseDate(*dateStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	migrateCanonicalPlays(paths, store)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays error:", err)
		os.Exit(1)
	}
	artistMetadata := genres.ArtistsForPlays(store, allPlays)

	generateDailyNote(allPlays, date, false, artistMetadata, noteOutputRoot(*outDir))
}

func weeklyNotePath(listeningDir string, date time.Time) string {
	monday, _ := render.WeekBounds(date)
	weekStr := render.WeekStr(monday)
	return filepath.Join(listeningDir, fmt.Sprintf("spotify-%s.md", weekStr))
}

func dailyNotePath(listeningDir string, date time.Time) string {
	dayStr := date.Local().Format("2006-01-02")
	return filepath.Join(listeningDir, fmt.Sprintf("spotify-%s.md", dayStr))
}

func missingWeeklyDates(listeningDir string, now time.Time, weeks int) []time.Time {
	missingDates := make([]time.Time, 0, weeks)
	for i := 0; i < weeks; i++ {
		weekDate := now.AddDate(0, 0, -(i * 7))
		notePath := weeklyNotePath(listeningDir, weekDate)
		if _, err := os.Stat(notePath); os.IsNotExist(err) {
			missingDates = append(missingDates, weekDate)
		}
	}
	return missingDates
}

func missingDailyDates(listeningDir string, now time.Time, totalDays int) []time.Time {
	missingDays := make([]time.Time, 0, totalDays)
	for i := 0; i < totalDays; i++ {
		day := now.AddDate(0, 0, -i)
		notePath := dailyNotePath(listeningDir, day)
		if _, err := os.Stat(notePath); os.IsNotExist(err) {
			missingDays = append(missingDays, day)
		}
	}
	return missingDays
}

func generateMissingWeeklyNotes(paths runtimePaths, missingDates []time.Time, outputRoot string) {
	if len(missingDates) == 0 {
		fmt.Println("Weekly notes: all caught up.")
		return
	}
	fmt.Printf("Found %d missing weekly note(s), generating...\n", len(missingDates))
	for i := len(missingDates) - 1; i >= 0; i-- {
		generateWeeklyNote(missingDates[i], paths, outputRoot)
	}
}

func generateMissingDailyNotes(allPlays []models.Play, missingDays []time.Time, totalDays int, artistMetadata map[string]genres.ArtistRecord, outputRoot string) {
	fmt.Printf("Checking %d days for missing daily notes...\n", totalDays)
	for _, day := range missingDays {
		generateDailyNote(allPlays, day, false, artistMetadata, outputRoot)
	}
}

func runCatchUp(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("catch-up", flag.ExitOnError)
	weeks := fs.Int("weeks", 8, "number of weeks to check")
	outDir := fs.String("out-dir", "", "output root override for safe sandbox generation")
	_ = fs.Parse(args)

	outputRoot := noteOutputRoot(*outDir)
	listeningDir := filepath.Join(outputRoot, "music", "listening")
	now := time.Now()

	missingWeeks := missingWeeklyDates(listeningDir, now, *weeks)
	generateMissingWeeklyNotes(paths, missingWeeks, outputRoot)

	store := loadGenreStore(paths)
	migrateCanonicalPlays(paths, store)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays error:", err)
		os.Exit(1)
	}
	artistMetadata := genres.ArtistsForPlays(store, allPlays)

	totalDays := *weeks * 7
	missingDays := missingDailyDates(listeningDir, now, totalDays)
	generateMissingDailyNotes(allPlays, missingDays, totalDays, artistMetadata, outputRoot)
	fmt.Println("Done.")
}

func runGenreBackfill(paths runtimePaths) {
	c, err := getClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	migrateCanonicalPlays(paths, store)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays error:", err)
		os.Exit(1)
	}

	if err := ensurePlaysDir(paths.genresPath); err != nil {
		fmt.Fprintln(os.Stderr, "data dir error:", err)
		os.Exit(1)
	}

	uncached := genres.UncachedArtistIDs(store, allPlays)
	fmt.Printf("Total plays: %d, cached artists: %d, uncached: %d\n", len(allPlays), len(store.Artists), len(uncached))

	if len(uncached) == 0 {
		fmt.Println("All artists already cached.")
	} else {
		fmt.Printf("Fetching genres for %d artist(s)...\n", len(uncached))
		artists, rawArtistBodies, err := fetch.GetArtistsBatchRaw(c, uncached)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fetch error:", err)
			os.Exit(1)
		}
		writeRawSpotifyArtists(paths, time.Now(), rawArtistBodies)
		for _, a := range artists {
			genres.CanonicalizeTopArtist(store, a)
		}
		saveGenreStore(paths, store)
		fmt.Printf("Cached %d artist(s).\n", len(artists))
	}

	// Update all existing artist stubs
	vault := vaultPath()
	updated := 0
	for _, entry := range genres.ArtistRecords(store) {
		if err := render.UpdateArtistStub(entry, vault); err != nil {
			fmt.Fprintf(os.Stderr, "warning: update artist stub for %s: %v\n", entry.Name, err)
		} else {
			updated++
		}
	}
	fmt.Printf("Updated %d artist stub(s).\n", updated)
}

func runImageBackfill(paths runtimePaths) {
	c, err := getClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Using metadata store: %s\n", paths.genresPath)

	store := loadGenreStore(paths)
	migrateCanonicalPlays(paths, store)

	missing := genres.MissingImagesArtistIDs(store)
	fmt.Printf("Artists missing images: %d\n", len(missing))
	if len(missing) == 0 {
		fmt.Println("All artists already have images.")
		return
	}

	artists, rawArtistBodies, err := fetch.GetArtistsBatchRaw(c, missing)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch error:", err)
		os.Exit(1)
	}
	writeRawSpotifyArtists(paths, time.Now(), rawArtistBodies)

	updatedNames := make([]string, 0, len(artists))
	for _, a := range artists {
		canonical := genres.CanonicalizeTopArtist(store, a)
		cachedName := canonical.Name
		genres.UpdateImages(store, a.ID, a.Images)
		if cachedName != "" && cachedName != a.Name {
			updatedNames = append(updatedNames, fmt.Sprintf("%s [id=%s, cached as %q]", a.Name, a.ID, cachedName))
		} else {
			updatedNames = append(updatedNames, fmt.Sprintf("%s [id=%s]", a.Name, a.ID))
		}
	}
	sort.Strings(updatedNames)

	saveGenreStore(paths, store)
	fmt.Printf("Updated images for %d artist(s).\n", len(artists))
	for _, name := range updatedNames {
		fmt.Printf("  - %s\n", name)
	}
}

func runSetlist(args []string) {
	fs := flag.NewFlagSet("setlist", flag.ExitOnError)
	dateStr := fs.String("date", "", "date in YYYY-MM-DD format (default: today)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: music-garden setlist <artist> [--date YYYY-MM-DD]")
		os.Exit(1)
	}
	artist := fs.Arg(0)

	date, err := parseDate(*dateStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dateFormatted := date.Format("2006-01-02")

	setlist, err := fetch.GetSetlist(artist, dateFormatted)
	if err != nil {
		fmt.Fprintln(os.Stderr, "setlist.fm error:", err)
		os.Exit(1)
	}

	fmt.Printf("%s — %s — %s\n", setlist.ArtistName, setlist.VenueName, setlist.CityName)
	fmt.Printf("%s\n\n", dateFormatted)

	for _, s := range setlist.Sets {
		if s.Name != "" {
			fmt.Printf("%s:\n", s.Name)
		} else {
			fmt.Printf("Set 1:\n")
		}
		for i, song := range s.Songs {
			fmt.Printf("%d. %s\n", i+1, song)
		}
		fmt.Println()
	}

	if setlist.URL != "" {
		fmt.Printf("Setlist.fm: %s\n", setlist.URL)
	}
}

func runMusicBrainzEnrichArtist(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("musicbrainz-enrich-artist", flag.ExitOnError)
	spotifyID := fs.String("spotify-id", "", "Spotify artist ID")
	name := fs.String("name", "", "artist display name")
	spotifyURL := fs.String("spotify-url", "", "Spotify artist URL")
	mbid := fs.String("mbid", "", "existing MusicBrainz artist ID override")
	_ = fs.Parse(args)

	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(os.Stderr, "--name is required")
		os.Exit(1)
	}

	mb, err := getMusicBrainzClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicbrainz client error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	record, err := enrichMusicBrainzArtist(paths, mb, store, mbnormalize.ArtistSeed{
		SpotifyArtistID: *spotifyID,
		Name:            *name,
		SpotifyURL:      *spotifyURL,
	}, strings.TrimSpace(*mbid))
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicbrainz artist enrichment error:", err)
		os.Exit(1)
	}
	saveGenreStore(paths, store)
	fmt.Printf("Enriched artist %q as %s (MusicBrainz %s)\n", record.Name, record.Slug, record.MusicBrainzArtistID)
}

func runMusicBrainzEnrichAlbum(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("musicbrainz-enrich-album", flag.ExitOnError)
	spotifyAlbumID := fs.String("spotify-album-id", "", "Spotify album ID")
	artistName := fs.String("artist", "", "primary artist name")
	artistSpotifyID := fs.String("artist-spotify-id", "", "Spotify artist ID")
	artistSpotifyURL := fs.String("artist-spotify-url", "", "Spotify artist URL")
	albumName := fs.String("name", "", "album or release name")
	mbid := fs.String("mbid", "", "existing MusicBrainz release-group ID override")
	_ = fs.Parse(args)

	if strings.TrimSpace(*artistName) == "" || strings.TrimSpace(*albumName) == "" {
		fmt.Fprintln(os.Stderr, "--artist and --name are required")
		os.Exit(1)
	}

	mb, err := getMusicBrainzClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicbrainz client error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	releaseRecord, err := enrichMusicBrainzAlbum(paths, mb, store, mbnormalize.ReleaseSeed{
		SpotifyAlbumID:       *spotifyAlbumID,
		Name:                 *albumName,
		PrimaryArtistName:    *artistName,
		PrimaryArtistID:      *artistSpotifyID,
		PrimaryArtistSpotify: *artistSpotifyURL,
	}, strings.TrimSpace(*mbid))
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicbrainz album enrichment error:", err)
		os.Exit(1)
	}
	saveGenreStore(paths, store)
	fmt.Printf("Enriched release %q as %s (release-group %s)\n", releaseRecord.Name, releaseRecord.Slug, releaseRecord.MusicBrainzReleaseGroupID)
}

func pickArtistMatch(seedName string, matches []mbclient.SearchArtistResult) mbclient.SearchArtistResult {
	needle := strings.ToLower(strings.TrimSpace(seedName))
	for _, match := range matches {
		if strings.ToLower(strings.TrimSpace(match.Name)) == needle {
			return match
		}
	}
	return matches[0]
}

func pickReleaseGroupMatch(seedArtist, seedRelease string, matches []mbclient.SearchReleaseGroupResult) mbclient.SearchReleaseGroupResult {
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

func enrichMusicBrainzArtist(paths runtimePaths, mb *mbclient.Client, store *genres.Store, seed mbnormalize.ArtistSeed, mbid string) (genres.ArtistRecord, error) {
	var artistResult mbclient.FetchResult[mbclient.Artist]
	var err error
	if strings.TrimSpace(mbid) != "" {
		artistResult, err = mb.GetArtistByID(strings.TrimSpace(mbid))
		if err != nil {
			return genres.ArtistRecord{}, err
		}
		stem := strings.TrimSpace(mbid)
		writeRawMusicBrainz(paths, "artists", stem, artistResult.Endpoint, artistResult.RequestURL, artistResult.FetchedAt, artistResult.Body, artistResult.FromCache)
	} else {
		searchResult, err := mb.SearchArtistByName(seed.Name)
		if err != nil {
			return genres.ArtistRecord{}, err
		}
		searchStem := genres.Slug(seed.Name)
		if searchStem == "" {
			searchStem = "artist-search"
		}
		writeRawMusicBrainz(paths, "artist-search", searchStem, searchResult.Endpoint, searchResult.RequestURL, searchResult.FetchedAt, searchResult.Body, searchResult.FromCache)
		if len(searchResult.Value) == 0 {
			return genres.ArtistRecord{}, fmt.Errorf("no MusicBrainz artist matches for %q", seed.Name)
		}
		match := pickArtistMatch(seed.Name, searchResult.Value)
		artistResult, err = mb.GetArtistByID(match.ID)
		if err != nil {
			return genres.ArtistRecord{}, err
		}
		writeRawMusicBrainz(paths, "artists", match.ID, artistResult.Endpoint, artistResult.RequestURL, artistResult.FetchedAt, artistResult.Body, artistResult.FromCache)
	}

	normalized, record := mbnormalize.NormalizeArtist(store, seed, artistResult.Value)
	if err := datalayer.WriteNormalizedMusicBrainzArtist(paths.dataRoot, record.Slug, normalized); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write normalized MusicBrainz artist: %v\n", err)
	}
	genreRecords := make([]datalayer.NormalizedGenreRecord, 0, len(normalized.SourceGenres))
	for _, sourceGenre := range normalized.SourceGenres {
		genreRecord := datalayer.NormalizedGenreRecord{
			Source:      "musicbrainz",
			SourceGenre: sourceGenre,
		}
		if canonical, ok := genres.CanonicalGenre(store, sourceGenre); ok {
			genreRecord.CanonicalGenreSlug = canonical
		}
		genreRecords = append(genreRecords, genreRecord)
	}
	if err := datalayer.WriteNormalizedMusicBrainzGenres(paths.dataRoot, genreRecords); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write normalized MusicBrainz genres: %v\n", err)
	}
	return record, nil
}

func enrichMusicBrainzAlbum(paths runtimePaths, mb *mbclient.Client, store *genres.Store, seed mbnormalize.ReleaseSeed, mbid string) (genres.ReleaseRecord, error) {
	var groupResult mbclient.FetchResult[mbclient.ReleaseGroup]
	var err error
	if strings.TrimSpace(mbid) != "" {
		groupResult, err = mb.GetReleaseGroupByID(strings.TrimSpace(mbid))
		if err != nil {
			return genres.ReleaseRecord{}, err
		}
		writeRawMusicBrainz(paths, "release-groups", strings.TrimSpace(mbid), groupResult.Endpoint, groupResult.RequestURL, groupResult.FetchedAt, groupResult.Body, groupResult.FromCache)
	} else {
		searchResult, err := mb.SearchReleaseGroups(seed.PrimaryArtistName, seed.Name)
		if err != nil {
			return genres.ReleaseRecord{}, err
		}
		searchStem := genres.Slug(seed.PrimaryArtistName) + "--" + genres.Slug(seed.Name)
		searchStem = strings.Trim(searchStem, "-")
		if searchStem == "" {
			searchStem = "release-group-search"
		}
		writeRawMusicBrainz(paths, "release-group-search", searchStem, searchResult.Endpoint, searchResult.RequestURL, searchResult.FetchedAt, searchResult.Body, searchResult.FromCache)
		if len(searchResult.Value) == 0 {
			return genres.ReleaseRecord{}, fmt.Errorf("no MusicBrainz release-group matches for %q by %q", seed.Name, seed.PrimaryArtistName)
		}
		match := pickReleaseGroupMatch(seed.PrimaryArtistName, seed.Name, searchResult.Value)
		groupResult, err = mb.GetReleaseGroupByID(match.ID)
		if err != nil {
			return genres.ReleaseRecord{}, err
		}
		writeRawMusicBrainz(paths, "release-groups", match.ID, groupResult.Endpoint, groupResult.RequestURL, groupResult.FetchedAt, groupResult.Body, groupResult.FromCache)
	}

	normalized, releaseRecord, _, genreRecords := mbnormalize.NormalizeRelease(store, seed, groupResult.Value)
	if err := datalayer.WriteNormalizedMusicBrainzRelease(paths.dataRoot, releaseRecord.Slug, normalized); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write normalized MusicBrainz release: %v\n", err)
	}
	if err := datalayer.WriteNormalizedMusicBrainzGenres(paths.dataRoot, genreRecords); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write normalized MusicBrainz genres: %v\n", err)
	}
	return releaseRecord, nil
}

func runWikipediaEnrichGenre(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("wikipedia-enrich-genre", flag.ExitOnError)
	slug := fs.String("slug", "", "canonical genre slug")
	mappingPath := fs.String("mapping", defaultGenreMappingPath(paths), "genre mapping JSON path")
	_ = fs.Parse(args)

	if strings.TrimSpace(*slug) == "" {
		fmt.Fprintln(os.Stderr, "--slug is required")
		os.Exit(1)
	}

	client, err := getMediaWikiClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mediawiki client error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	mapping, err := loadGenrePageMapping(*mappingPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre mapping error:", err)
		os.Exit(1)
	}
	seed := mapping[strings.TrimSpace(*slug)]
	seed.CanonicalSlug = strings.TrimSpace(*slug)
	if seed.SearchTerm == "" {
		seed.SearchTerm = strings.ReplaceAll(seed.CanonicalSlug, "-", " ")
	}
	status, title, err := enrichWikipediaGenre(paths, client, store, seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wikipedia genre enrichment error:", err)
		os.Exit(1)
	}
	saveGenreStore(paths, store)
	switch status {
	case "matched":
		fmt.Printf("Enriched genre %q with %q\n", seed.CanonicalSlug, title)
	case "ambiguous":
		fmt.Printf("Genre %q resolved to a disambiguation page; add a page_title override to %s\n", seed.CanonicalSlug, *mappingPath)
	default:
		fmt.Printf("Genre %q not auto-resolved (%s)\n", seed.CanonicalSlug, status)
	}
}

func runWikipediaEnrichArtist(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("wikipedia-enrich-artist", flag.ExitOnError)
	slug := fs.String("slug", "", "canonical artist slug")
	mappingPath := fs.String("mapping", defaultArtistMappingPath(paths), "artist mapping JSON path")
	_ = fs.Parse(args)

	if strings.TrimSpace(*slug) == "" {
		fmt.Fprintln(os.Stderr, "--slug is required")
		os.Exit(1)
	}

	client, err := getMediaWikiClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mediawiki client error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	artist, ok := store.Artists[strings.TrimSpace(*slug)]
	if !ok {
		fmt.Fprintf(os.Stderr, "artist slug not found in canonical store: %s\n", strings.TrimSpace(*slug))
		os.Exit(1)
	}

	mapping, err := loadArtistPageMapping(*mappingPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "artist mapping error:", err)
		os.Exit(1)
	}
	seed := mapping[strings.TrimSpace(*slug)]
	seed.CanonicalSlug = strings.TrimSpace(*slug)
	if strings.TrimSpace(seed.Name) == "" {
		seed.Name = artist.Name
	}
	if seed.SearchTerm == "" {
		seed.SearchTerm = artist.Name
	}

	status, title, err := enrichWikipediaArtist(paths, client, store, seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wikipedia artist enrichment error:", err)
		os.Exit(1)
	}
	saveGenreStore(paths, store)
	switch status {
	case "matched":
		fmt.Printf("Enriched artist %q with %q\n", seed.CanonicalSlug, title)
	case "ambiguous":
		fmt.Printf("Artist %q resolved to a disambiguation page; add a page_title override to %s\n", seed.CanonicalSlug, *mappingPath)
	default:
		fmt.Printf("Artist %q not auto-resolved (%s)\n", seed.CanonicalSlug, status)
	}
}

func defaultGenreMappingPath(paths runtimePaths) string {
	cwdPath := filepath.Join(paths.cwd, "data", "genre-page-mapping.json")
	if info, err := os.Stat(cwdPath); err == nil && !info.IsDir() {
		return cwdPath
	}
	return filepath.Join(paths.dataRoot, "genre-page-mapping.json")
}

func enrichWikipediaGenre(paths runtimePaths, client *mwclient.Client, store *genres.Store, seed mwnormalize.GenreSeed) (string, string, error) {
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
		writeRawWikipedia(paths, "search", searchStem, search.Endpoint, search.RequestURL, search.FetchedAt, search.Body, search.FromCache)
		candidates = make([]string, 0, len(search.Value))
		for _, candidate := range search.Value {
			candidates = append(candidates, candidate.Title)
		}
		pageTitle, err = chooseWikipediaPageTitle(seed, search.Value)
		if err != nil {
			status := unresolvedStatus(search.Value)
			normalized, record := mwnormalize.NormalizeUnresolvedGenre(seed, status, candidates, fetchedAt)
			_ = datalayer.WriteNormalizedWikipediaGenre(paths.dataRoot, seed.CanonicalSlug, normalized)
			genres.UpsertGenreEditorial(store, record)
			return status, "", nil
		}
	}

	summary, err := client.GetSummary(pageTitle)
	if err != nil {
		return "", "", err
	}
	summaryStem := genres.Slug(seed.CanonicalSlug)
	writeRawWikipedia(paths, "summaries", summaryStem, summary.Endpoint, summary.RequestURL, summary.FetchedAt, summary.Body, summary.FromCache)
	if summary.Value.Type == "disambiguation" {
		normalized, record := mwnormalize.NormalizeUnresolvedGenre(seed, "ambiguous", candidates, fetchedAt)
		_ = datalayer.WriteNormalizedWikipediaGenre(paths.dataRoot, seed.CanonicalSlug, normalized)
		genres.UpsertGenreEditorial(store, record)
		return "ambiguous", "", nil
	}

	imageInfos := collectWikipediaImageCandidates(paths, client, pageTitle, summaryStem)
	normalized, record := mwnormalize.NormalizeMatchedGenre(seed, summary.Value, candidates, imageInfos, fetchedAt)
	if err := datalayer.WriteNormalizedWikipediaGenre(paths.dataRoot, seed.CanonicalSlug, normalized); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write normalized wikipedia genre: %v\n", err)
	}
	genres.UpsertGenreEditorial(store, record)
	return "matched", record.WikipediaTitle, nil
}

func enrichWikipediaArtist(paths runtimePaths, client *mwclient.Client, store *genres.Store, seed mwnormalize.ArtistSeed) (string, string, error) {
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
		writeRawWikipedia(paths, "search", "artist--"+searchStem, search.Endpoint, search.RequestURL, search.FetchedAt, search.Body, search.FromCache)
		candidates = make([]string, 0, len(search.Value))
		for _, candidate := range search.Value {
			candidates = append(candidates, candidate.Title)
		}
		pageTitle, err = chooseWikipediaArtistPageTitle(seed, search.Value)
		if err != nil {
			status := unresolvedStatus(search.Value)
			normalized, record := mwnormalize.NormalizeUnresolvedArtist(seed, status, candidates, fetchedAt)
			if err := datalayer.WriteNormalizedWikipediaArtist(paths.dataRoot, seed.CanonicalSlug, normalized); err != nil {
				fmt.Fprintf(os.Stderr, "warning: write normalized wikipedia artist: %v\n", err)
			}
			genres.UpsertArtistEditorial(store, seed.CanonicalSlug, record)
			return status, "", nil
		}
	}

	summary, err := client.GetSummary(pageTitle)
	if err != nil {
		return "", "", err
	}
	summaryStem := genres.Slug(seed.CanonicalSlug)
	writeRawWikipedia(paths, "summaries", "artist--"+summaryStem, summary.Endpoint, summary.RequestURL, summary.FetchedAt, summary.Body, summary.FromCache)
	if summary.Value.Type == "disambiguation" {
		normalized, record := mwnormalize.NormalizeUnresolvedArtist(seed, "ambiguous", candidates, fetchedAt)
		if err := datalayer.WriteNormalizedWikipediaArtist(paths.dataRoot, seed.CanonicalSlug, normalized); err != nil {
			fmt.Fprintf(os.Stderr, "warning: write normalized wikipedia artist: %v\n", err)
		}
		genres.UpsertArtistEditorial(store, seed.CanonicalSlug, record)
		return "ambiguous", "", nil
	}

	imageInfos := collectWikipediaImageCandidates(paths, client, pageTitle, "artist--"+summaryStem)
	normalized, record := mwnormalize.NormalizeMatchedArtist(seed, summary.Value, candidates, imageInfos, fetchedAt)
	if err := datalayer.WriteNormalizedWikipediaArtist(paths.dataRoot, seed.CanonicalSlug, normalized); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write normalized wikipedia artist: %v\n", err)
	}
	genres.UpsertArtistEditorial(store, seed.CanonicalSlug, record)
	return "matched", record.WikipediaTitle, nil
}

func defaultGenreTaxonomyPath(paths runtimePaths) string {
	cwdPath := filepath.Join(paths.cwd, "data", "genre-taxonomy.json")
	if info, err := os.Stat(cwdPath); err == nil && !info.IsDir() {
		return cwdPath
	}
	return filepath.Join(paths.dataRoot, "genre-taxonomy.json")
}

func defaultArtistMappingPath(paths runtimePaths) string {
	cwdPath := filepath.Join(paths.cwd, "data", "artist-page-mapping.json")
	if info, err := os.Stat(cwdPath); err == nil && !info.IsDir() {
		return cwdPath
	}
	return filepath.Join(paths.dataRoot, "artist-page-mapping.json")
}

func loadGenrePageMapping(path string) (map[string]mwnormalize.GenreSeed, error) {
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

func loadArtistPageMapping(path string) (map[string]mwnormalize.ArtistSeed, error) {
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

func canonicalGenreSlugsForBackfill(store *genres.Store) []string {
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

func chooseWikipediaPageTitle(seed mwnormalize.GenreSeed, candidates []mwclient.SearchResult) (string, error) {
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
		if strings.EqualFold(candidate.Title, seed.SearchTerm) {
			return candidate.Title, nil
		}
	}
	return "", fmt.Errorf("ambiguous search results")
}

func chooseWikipediaArtistPageTitle(seed mwnormalize.ArtistSeed, candidates []mwclient.SearchResult) (string, error) {
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

func unresolvedStatus(candidates []mwclient.SearchResult) string {
	if len(candidates) == 0 {
		return "not_found"
	}
	return "ambiguous"
}

func runGenreReport(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("genre-report", flag.ExitOnError)
	taxonomyPath := fs.String("taxonomy", defaultGenreTaxonomyPath(paths), "canonical genre taxonomy path")
	_ = fs.Parse(args)

	taxonomy, err := genres.LoadTaxonomy(*taxonomyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre taxonomy error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	report := genres.BuildGenreReport(taxonomy, store.PendingGenreAliases)
	fmt.Print(report.ReportString())
}

func runAggregateGenre(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("aggregate-genre", flag.ExitOnError)
	slug := fs.String("slug", "", "canonical genre slug")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genre flags:", err)
		os.Exit(1)
	}
	genreSlug := genres.Slug(*slug)
	if genreSlug == "" {
		fmt.Fprintln(os.Stderr, "aggregate genre error: --slug is required")
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genre load plays error:", err)
		os.Exit(1)
	}
	allPlays, _ = genres.ResolvePlays(store, allPlays)
	path, err := datalayer.RebuildAggregatedGenre(paths.dataRoot, store, allPlays, genreSlug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genre error:", err)
		os.Exit(1)
	}
	fmt.Printf("Aggregated genre %q -> %s\n", genreSlug, path)
}

func runAggregateGenres(paths runtimePaths) {
	store := loadGenreStore(paths)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genres load plays error:", err)
		os.Exit(1)
	}
	allPlays, _ = genres.ResolvePlays(store, allPlays)
	count, err := datalayer.RebuildAllAggregatedGenres(paths.dataRoot, store, allPlays)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregate genres error:", err)
		os.Exit(1)
	}
	fmt.Printf("Rebuilt %d aggregated genre record(s)\n", count)
}

func runGenerateGenrePages(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("generate-genre-pages", flag.ExitOnError)
	outDir := fs.String("out-dir", filepath.Join(paths.cwd, "content", "genres"), "output directory for generated genre markdown")
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
	updated, unchanged, err := render.WriteGenrePages(records, *outDir, tmplPath, selected)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate genre pages error:", err)
		os.Exit(1)
	}
	fmt.Printf("Generated genre pages in %s (updated=%d unchanged=%d)\n", *outDir, updated, unchanged)
}

func runMusicBrainzBackfillArtists(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("musicbrainz-backfill-artists", flag.ExitOnError)
	limit := fs.Int("limit", 0, "max artists to enrich")
	refresh := fs.Bool("refresh", false, "refresh artists that already have MusicBrainz IDs")
	_ = fs.Parse(args)

	mb, err := getMusicBrainzClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicbrainz client error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	records := genres.ArtistRecords(store)
	processed, skipped, failed, attempted := 0, 0, 0, 0
	for _, record := range records {
		if !*refresh && record.MusicBrainzArtistID != "" {
			skipped++
			continue
		}
		if strings.TrimSpace(record.Name) == "" {
			skipped++
			continue
		}
		if *limit > 0 && attempted >= *limit {
			break
		}
		attempted++
		_, err := enrichMusicBrainzArtist(paths, mb, store, mbnormalize.ArtistSeed{
			SpotifyArtistID: record.SpotifyArtistID,
			Name:            record.Name,
			SpotifyURL:      record.SpotifyURL,
		}, record.MusicBrainzArtistID)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "warning: musicbrainz artist %s: %v\n", record.Slug, err)
			continue
		}
		processed++
		fmt.Printf("MusicBrainz artist: %s\n", record.Slug)
	}
	saveGenreStore(paths, store)
	fmt.Printf("MusicBrainz artist backfill complete: processed=%d skipped=%d failed=%d\n", processed, skipped, failed)
}

func runMusicBrainzBackfillAlbums(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("musicbrainz-backfill-albums", flag.ExitOnError)
	limit := fs.Int("limit", 0, "max releases to enrich")
	refresh := fs.Bool("refresh", false, "refresh releases that already have MusicBrainz release-group IDs")
	_ = fs.Parse(args)

	mb, err := getMusicBrainzClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicbrainz client error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	releases := genres.ReleaseRecords(store)
	processed, skipped, failed, attempted := 0, 0, 0, 0
	for _, record := range releases {
		if !*refresh && record.MusicBrainzReleaseGroupID != "" {
			skipped++
			continue
		}
		artist, ok := store.Artists[record.PrimaryArtistSlug]
		if !ok || strings.TrimSpace(record.Name) == "" || strings.TrimSpace(artist.Name) == "" {
			skipped++
			continue
		}
		if *limit > 0 && attempted >= *limit {
			break
		}
		attempted++
		_, err := enrichMusicBrainzAlbum(paths, mb, store, mbnormalize.ReleaseSeed{
			SpotifyAlbumID:       record.SpotifyAlbumID,
			Name:                 record.Name,
			PrimaryArtistName:    artist.Name,
			PrimaryArtistID:      artist.SpotifyArtistID,
			PrimaryArtistSpotify: artist.SpotifyURL,
		}, record.MusicBrainzReleaseGroupID)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "warning: musicbrainz release %s: %v\n", record.Slug, err)
			continue
		}
		processed++
		fmt.Printf("MusicBrainz release: %s\n", record.Slug)
	}
	saveGenreStore(paths, store)
	fmt.Printf("MusicBrainz release backfill complete: processed=%d skipped=%d failed=%d\n", processed, skipped, failed)
}

func runWikipediaBackfillGenres(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("wikipedia-backfill-genres", flag.ExitOnError)
	limit := fs.Int("limit", 0, "max genres to enrich")
	refresh := fs.Bool("refresh", false, "refresh genres that already have matched Wikipedia metadata")
	mappingPath := fs.String("mapping", defaultGenreMappingPath(paths), "genre mapping JSON path")
	_ = fs.Parse(args)

	client, err := getMediaWikiClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mediawiki client error:", err)
		os.Exit(1)
	}
	mapping, err := loadGenrePageMapping(*mappingPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genre mapping error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	slugs := canonicalGenreSlugsForBackfill(store)
	processed, skipped, failed, attempted := 0, 0, 0, 0
	for _, slug := range slugs {
		if record, ok := genres.GenreEditorial(store, slug); ok && !*refresh && record.Status == "matched" && record.WikipediaURL != "" {
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
		status, _, err := enrichWikipediaGenre(paths, client, store, seed)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "warning: wikipedia genre %s: %v\n", slug, err)
			continue
		}
		processed++
		fmt.Printf("Wikipedia genre: %s (%s)\n", slug, status)
	}
	saveGenreStore(paths, store)
	fmt.Printf("Wikipedia genre backfill complete: processed=%d skipped=%d failed=%d\n", processed, skipped, failed)
}

func runWikipediaBackfillArtists(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("wikipedia-backfill-artists", flag.ExitOnError)
	limit := fs.Int("limit", 0, "max artists to enrich")
	refresh := fs.Bool("refresh", false, "refresh artists that already have matched Wikipedia metadata")
	mappingPath := fs.String("mapping", defaultArtistMappingPath(paths), "artist mapping JSON path")
	_ = fs.Parse(args)

	client, err := getMediaWikiClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mediawiki client error:", err)
		os.Exit(1)
	}
	mapping, err := loadArtistPageMapping(*mappingPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "artist mapping error:", err)
		os.Exit(1)
	}

	store := loadGenreStore(paths)
	records := genres.ArtistRecords(store)
	processed, skipped, failed, attempted := 0, 0, 0, 0
	for _, artist := range records {
		if !*refresh && artist.Status == "matched" && artist.WikipediaURL != "" {
			skipped++
			continue
		}
		if strings.TrimSpace(artist.Name) == "" {
			skipped++
			continue
		}
		if *limit > 0 && attempted >= *limit {
			break
		}
		attempted++
		seed := mapping[artist.Slug]
		seed.CanonicalSlug = artist.Slug
		if strings.TrimSpace(seed.Name) == "" {
			seed.Name = artist.Name
		}
		if seed.SearchTerm == "" {
			seed.SearchTerm = artist.Name
		}
		status, _, err := enrichWikipediaArtist(paths, client, store, seed)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "warning: wikipedia artist %s: %v\n", artist.Slug, err)
			continue
		}
		processed++
		fmt.Printf("Wikipedia artist: %s (%s)\n", artist.Slug, status)
	}
	saveGenreStore(paths, store)
	fmt.Printf("Wikipedia artist backfill complete: processed=%d skipped=%d failed=%d\n", processed, skipped, failed)
}

func collectWikipediaImageCandidates(paths runtimePaths, client *mwclient.Client, pageTitle, stem string) []mwclient.ImageInfo {
	var imageInfos []mwclient.ImageInfo
	pageImages, err := client.GetPageImages(pageTitle)
	if err != nil {
		return nil
	}
	writeRawWikipedia(paths, "page-images", stem, pageImages.Endpoint, pageImages.RequestURL, pageImages.FetchedAt, pageImages.Body, pageImages.FromCache)
	for _, fileTitle := range pageImages.Value {
		if !strings.HasPrefix(fileTitle, "File:") || !isLikelyGenreEditorialImage(fileTitle) {
			continue
		}
		info, infoErr := client.GetCommonsImageInfo(fileTitle)
		if infoErr != nil {
			continue
		}
		if !isLikelyGenreEditorialImage(info.Value.FileTitle) {
			continue
		}
		writeRawWikipedia(paths, "commons-images", stem+"--"+genres.Slug(fileTitle), info.Endpoint, info.RequestURL, info.FetchedAt, info.Body, info.FromCache)
		imageInfos = append(imageInfos, info.Value)
		if len(imageInfos) >= 5 {
			break
		}
	}
	return imageInfos
}

func isLikelyGenreEditorialImage(fileTitle string) bool {
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

func runPersona(args []string, paths runtimePaths) {
	fs := flag.NewFlagSet("persona", flag.ExitOnError)
	outDir := fs.String("out-dir", "", "output root override for safe sandbox generation")
	_ = fs.Parse(args)

	outputRoot := noteOutputRoot(*outDir)
	outPath := filepath.Join(outputRoot, "01-ai-brain", "context-packs", "Music Taste.md")
	tmplPath := filepath.Join(templatesDir(), "persona.md.tmpl")

	c, err := getClient(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("Fetching top artists (short_term, medium_term, long_term)...")
	fetchedAt := time.Now()
	topArtistsShort, rawTopArtistsShort, err := fetch.GetTopArtistsRaw(c, "short_term")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: top artists short_term: %v\n", err)
	}
	writeRawSpotifyTopArtists(paths, fetchedAt, "short_term", rawTopArtistsShort)
	topArtistsMedium, rawTopArtistsMedium, err := fetch.GetTopArtistsRaw(c, "medium_term")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: top artists medium_term: %v\n", err)
	}
	writeRawSpotifyTopArtists(paths, fetchedAt, "medium_term", rawTopArtistsMedium)
	topArtistsLong, rawTopArtistsLong, err := fetch.GetTopArtistsRaw(c, "long_term")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: top artists long_term: %v\n", err)
	}
	writeRawSpotifyTopArtists(paths, fetchedAt, "long_term", rawTopArtistsLong)

	store := loadGenreStore(paths)
	migrateCanonicalPlays(paths, store)
	allPlays, err := plays.LoadSharded(paths.playsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load plays error:", err)
		os.Exit(1)
	}
	weekPlays := render.PlaysForWeek(allPlays, time.Now())
	topArtistsShort = canonicalizeTopArtists(store, topArtistsShort)
	topArtistsMedium = canonicalizeTopArtists(store, topArtistsMedium)
	topArtistsLong = canonicalizeTopArtists(store, topArtistsLong)
	allTopArtists := append(append(topArtistsShort, topArtistsMedium...), topArtistsLong...)
	saveGenreStore(paths, store)

	// Update artist stubs with canonical metadata
	for _, a := range allTopArtists {
		record := genres.ArtistRecord{
			Name:                a.Name,
			Slug:                a.ArtistSlug,
			SpotifyArtistID:     a.ID,
			MusicBrainzArtistID: a.MusicBrainzArtistID,
			SpotifyURL:          a.SpotifyURL,
			Genres:              a.Genres,
			SourceGenres:        a.SourceGenres,
			Images:              a.Images,
		}
		if err := render.UpdateArtistStub(record, outputRoot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: update artist stub for %s: %v\n", a.Name, err)
		}
	}

	content, err := render.RenderPersona(topArtistsShort, topArtistsMedium, topArtistsLong, weekPlays, tmplPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render error:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir error:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
		os.Exit(1)
	}

	fmt.Println("Written:", outPath)
}

func runDoctor(paths runtimePaths) int {
	issues := 0

	fmt.Println("Runtime configuration")
	fmt.Println("---------------------")
	fmt.Println("Working directory:", paths.cwd)
	fmt.Println("Executable:", executablePath())
	if paths.stateDir == "" {
		fmt.Println("MUSIC_STATE_DIR: (not set)")
	} else {
		fmt.Println("MUSIC_STATE_DIR:", paths.stateDir)
	}

	printPathStatus("Dotenv path", paths.dotEnvPath, true)
	printPathStatus("Tokens path", paths.tokensPath, false)
	printPathStatus("Plays dir", paths.playsDir, false)
	printPathStatus("Plays legacy", paths.playsPath, true)
	printPathStatus("Genres path", paths.genresPath, false)
	printPathStatus("Data root", paths.dataRoot, false)

	if paths.playsOverride {
		fmt.Println("MUSIC_PLAYS_DIR override:", paths.playsDir)
	}
	if paths.genresOverride {
		fmt.Println("MUSIC_GENRES_PATH override:", paths.genresPath)
	}

	templates := templatesDir()
	printPathStatus("Templates dir", templates, false)

	vault := strings.TrimSpace(os.Getenv("OBSIDIAN_VAULT_PATH"))
	if vault == "" {
		fmt.Println("Vault path: (OBSIDIAN_VAULT_PATH not set)")
		issues++
	} else {
		listeningDir := filepath.Join(vault, "music", "listening")
		fmt.Println("Vault path:", vault)
		fmt.Println("Listening output:", listeningDir)
	}

	if paths.dotEnvFallback {
		issues++
		fmt.Printf("Warning: using CWD fallback for .env (%s) because %s is missing\n", paths.dotEnvPath, filepath.Join(paths.stateDir, ".env"))
	}
	if paths.tokensFallback {
		issues++
		fmt.Printf("Warning: using CWD fallback for tokens.json (%s) because %s is missing\n", paths.tokensPath, filepath.Join(paths.stateDir, "tokens.json"))
	}
	if paths.playsFallback {
		issues++
		fmt.Printf("Warning: using CWD fallback for plays dir (%s) because %s is missing\n", paths.playsDir, filepath.Join(paths.stateDir, "data", "plays"))
	}

	if strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID")) == "" {
		fmt.Println("Warning: SPOTIFY_CLIENT_ID is not set")
		issues++
	}
	if strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET")) == "" {
		fmt.Println("Warning: SPOTIFY_CLIENT_SECRET is not set")
		issues++
	}

	collectLabel, weeklyLabel, collectLog, weeklyLog := launchdDefaults()
	fmt.Println("Launchd collect label:", collectLabel)
	fmt.Println("Launchd weekly label:", weeklyLabel)
	fmt.Println("Collect log path:", collectLog)
	fmt.Println("Weekly log path:", weeklyLog)

	if status, err := launchdJobStatus(collectLabel); err == nil {
		fmt.Println("Collect job status:", status)
	} else {
		fmt.Println("Collect job status: unknown (", err, ")")
	}
	if status, err := launchdJobStatus(weeklyLabel); err == nil {
		fmt.Println("Weekly job status:", status)
	} else {
		fmt.Println("Weekly job status: unknown (", err, ")")
	}

	if issues > 0 {
		fmt.Printf("\nDoctor found %d issue(s).\n", issues)
		return 1
	}
	fmt.Println("\nDoctor found no issues.")
	return 0
}

func executablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "(unknown)"
	}
	return exe
}

func printPathStatus(label, path string, optional bool) {
	fi, err := os.Stat(path)
	if err == nil {
		kind := "file"
		if fi.IsDir() {
			kind = "dir"
		}
		fmt.Printf("%s: %s (%s, present)\n", label, path, kind)
		return
	} else if os.IsNotExist(err) {
		if optional {
			fmt.Printf("%s: %s (missing, optional)\n", label, path)
		} else {
			fmt.Printf("%s: %s (missing)\n", label, path)
		}
		return
	}
	fmt.Printf("%s: %s (error: %v)\n", label, path, err)
}

func launchdDefaults() (collectLabel, weeklyLabel, collectLog, weeklyLog string) {
	user := strings.TrimSpace(os.Getenv("USER"))
	if user == "" {
		user = "unknown"
	}

	collectLabel = os.Getenv("MUSIC_COLLECT_SPOTIFY_LABEL")
	if collectLabel == "" {
		collectLabel = fmt.Sprintf("com.%s.music-collect-spotify", user)
	}
	weeklyLabel = os.Getenv("MUSIC_WEEKLY_SPOTIFY_LABEL")
	if weeklyLabel == "" {
		weeklyLabel = fmt.Sprintf("com.%s.music-weekly-spotify", user)
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library", "Application Support", "music-garden", "logs")
	collectLog = filepath.Join(logDir, "collect.log")
	weeklyLog = filepath.Join(logDir, "weekly.log")

	return collectLabel, weeklyLabel, collectLog, weeklyLog
}

func launchdJobStatus(label string) (string, error) {
	if _, err := exec.LookPath("launchctl"); err != nil {
		return "", fmt.Errorf("launchctl not found")
	}
	cmd := exec.Command("launchctl", "list", label)
	if err := cmd.Run(); err != nil {
		return "not loaded", nil
	}
	return "loaded", nil
}
