package publicexport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
	"github.com/benstraw/music-garden/internal/models"
	"github.com/benstraw/music-garden/internal/reviews"
)

const ContractVersion = "1.0.0"

type Options struct {
	OutputDir      string
	SourceRevision string
	Timezone       string
	Catalog        *genres.Catalog
	Plays          []models.Play
	Reviews        *reviews.Store
}

type Publication struct {
	Indexed bool     `json:"indexed"`
	Reasons []string `json:"reasons"`
}

type TrackStat struct {
	Name        string `json:"name"`
	Artist      string `json:"artist,omitempty"`
	ArtistSlug  string `json:"artist_slug,omitempty"`
	Album       string `json:"album,omitempty"`
	SpotifyURL  string `json:"spotify_url,omitempty"`
	Plays       int    `json:"plays"`
	DiscNumber  int    `json:"disc_number,omitempty"`
	TrackNumber int    `json:"track_number,omitempty"`
}

type Artist struct {
	Slug             string                         `json:"slug"`
	Name             string                         `json:"name"`
	SortName         string                         `json:"sort_name,omitempty"`
	ArtistType       string                         `json:"artist_type,omitempty"`
	Country          string                         `json:"country,omitempty"`
	LifeSpan         *genres.LifeSpan               `json:"life_span,omitempty"`
	Aliases          []string                       `json:"aliases,omitempty"`
	SpotifyID        string                         `json:"spotify_id,omitempty"`
	MusicBrainzID    string                         `json:"musicbrainz_id,omitempty"`
	WikidataID       string                         `json:"wikidata_id,omitempty"`
	SpotifyURL       string                         `json:"spotify_url,omitempty"`
	Genres           []string                       `json:"genres,omitempty"`
	Summary          string                         `json:"summary,omitempty"`
	Attribution      *genres.GenreSourceAttribution `json:"attribution,omitempty"`
	Hero             *genres.GenreImageRecord       `json:"hero,omitempty"`
	Gallery          []genres.GenreImageRecord      `json:"gallery,omitempty"`
	SpotifyImages    []models.ArtistImage           `json:"spotify_images,omitempty"`
	Relationships    []genres.ArtistRelationship    `json:"relationships,omitempty"`
	TotalPlays       int                            `json:"total_plays"`
	FirstSeen        string                         `json:"first_seen,omitempty"`
	LastSeen         string                         `json:"last_seen,omitempty"`
	ListeningMonths  map[string]int                 `json:"listening_months,omitempty"`
	TopTracks        []TrackStat                    `json:"top_tracks,omitempty"`
	ListenedReleases []string                       `json:"listened_releases,omitempty"`
	Publication      Publication                    `json:"publication"`
}

type GenreNeighbor struct {
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	Score         float64 `json:"score"`
	SharedArtists int     `json:"shared_artists"`
}

type Genre struct {
	Slug              string                         `json:"slug"`
	Name              string                         `json:"name"`
	ParentSlug        string                         `json:"parent_slug,omitempty"`
	ChildSlugs        []string                       `json:"child_slugs,omitempty"`
	Aliases           []string                       `json:"aliases,omitempty"`
	Summary           string                         `json:"summary,omitempty"`
	Attribution       *genres.GenreSourceAttribution `json:"attribution,omitempty"`
	Hero              *genres.GenreImageRecord       `json:"hero,omitempty"`
	PlayCount         int                            `json:"play_count"`
	UniqueArtistCount int                            `json:"unique_artist_count"`
	FirstSeen         string                         `json:"first_seen,omitempty"`
	LastSeen          string                         `json:"last_seen,omitempty"`
	TopArtists        []EntityStat                   `json:"top_artists,omitempty"`
	TopTracks         []TrackStat                    `json:"top_tracks,omitempty"`
	ListenedReleases  []EntityStat                   `json:"listened_releases,omitempty"`
	Neighbors         []GenreNeighbor                `json:"listening_neighbors,omitempty"`
	Publication       Publication                    `json:"publication"`
}

type EntityStat struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Plays int    `json:"plays"`
}

type Release struct {
	Slug                      string                  `json:"slug"`
	Name                      string                  `json:"name"`
	PrimaryArtistSlug         string                  `json:"primary_artist_slug,omitempty"`
	PrimaryArtistName         string                  `json:"primary_artist_name,omitempty"`
	SpotifyAlbumID            string                  `json:"spotify_album_id,omitempty"`
	MusicBrainzReleaseGroupID string                  `json:"musicbrainz_release_group_id,omitempty"`
	PrimaryType               string                  `json:"primary_type,omitempty"`
	SecondaryTypes            []string                `json:"secondary_types,omitempty"`
	FirstReleaseDate          string                  `json:"first_release_date,omitempty"`
	ArtistCredits             []genres.PlayArtistRef  `json:"artist_credits,omitempty"`
	Editions                  []genres.ReleaseEdition `json:"editions"`
	Sessions                  []AlbumSession          `json:"sessions"`
	FirstListened             string                  `json:"first_listened"`
	LastListened              string                  `json:"last_listened"`
	TrackEvidence             []TrackStat             `json:"track_evidence,omitempty"`
	Publication               Publication             `json:"publication"`
}

type Week struct {
	WeekStart       string               `json:"weekStart"`
	WeekEnd         string               `json:"weekEnd"`
	TotalPlays      int                  `json:"total_plays"`
	UniqueTracks    int                  `json:"unique_tracks"`
	UniqueArtists   int                  `json:"unique_artists"`
	UniqueAlbums    int                  `json:"unique_albums"`
	TotalMinutes    int                  `json:"total_minutes"`
	TopArtistName   string               `json:"topArtistName,omitempty"`
	TopArtistImages []models.ArtistImage `json:"topArtistImages,omitempty"`
	TopArtists      []EntityStat         `json:"top_artists,omitempty"`
	TopTracks       []TrackStat          `json:"top_tracks,omitempty"`
	TopGenres       []EntityStat         `json:"top_genres,omitempty"`
}

type Overview struct {
	PublishedThrough string `json:"published_through,omitempty"`
	ArtistCount      int    `json:"artist_count"`
	IndexedArtists   int    `json:"indexed_artists"`
	IndexedGenres    int    `json:"indexed_genres"`
	ListenedReleases int    `json:"listened_releases"`
	WeeklyPages      int    `json:"weekly_pages"`
}

type Redirects struct {
	Artists map[string]string `json:"artists"`
	Genres  map[string]string `json:"genres"`
}

type FileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Contract         string       `json:"contract"`
	ContractVersion  string       `json:"contract_version"`
	Privacy          string       `json:"privacy"`
	Timezone         string       `json:"timezone"`
	PublishedThrough string       `json:"published_through,omitempty"`
	SourceRevision   string       `json:"source_revision,omitempty"`
	DatasetSHA256    string       `json:"dataset_sha256"`
	Counts           Overview     `json:"counts"`
	Files            []FileDigest `json:"files"`
}

type builtData struct {
	artists   map[string]Artist
	genres    map[string]Genre
	releases  map[string]Release
	weeks     map[string]Week
	redirects Redirects
	overview  Overview
}

func Export(opts Options) (Manifest, error) {
	if opts.Catalog == nil {
		return Manifest{}, fmt.Errorf("catalog is required")
	}
	if opts.Reviews == nil {
		opts.Reviews, _ = reviews.Load("")
	}
	loc, err := time.LoadLocation(opts.Timezone)
	if err != nil {
		return Manifest{}, fmt.Errorf("load timezone %q: %w", opts.Timezone, err)
	}
	resolved, _ := genres.ResolvePlays(opts.Catalog, opts.Plays)
	data := buildData(resolved, opts.Catalog, opts.Reviews, loc)

	parent := filepath.Dir(opts.OutputDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("create export parent: %w", err)
	}
	tmp, err := os.MkdirTemp(parent, ".public-v1-")
	if err != nil {
		return Manifest{}, fmt.Errorf("create temporary export: %w", err)
	}
	defer os.RemoveAll(tmp)

	files := map[string]any{
		"overview.json":  data.overview,
		"artists.json":   data.artists,
		"genres.json":    data.genres,
		"releases.json":  data.releases,
		"redirects.json": data.redirects,
	}
	for weekKey, week := range data.weeks {
		year := strings.SplitN(weekKey, "-", 2)[0]
		files[filepath.ToSlash(filepath.Join("weeks", year, weekKey+".json"))] = week
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	digests := make([]FileDigest, 0, len(paths))
	datasetHasher := sha256.New()
	for _, path := range paths {
		encoded, err := marshalDeterministic(files[path])
		if err != nil {
			return Manifest{}, fmt.Errorf("encode %s: %w", path, err)
		}
		if strings.Contains(string(encoded), `"played_at"`) {
			return Manifest{}, fmt.Errorf("privacy validation failed: %s contains played_at", path)
		}
		fullPath := filepath.Join(tmp, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return Manifest{}, err
		}
		if err := os.WriteFile(fullPath, encoded, 0o644); err != nil {
			return Manifest{}, err
		}
		sum := sha256.Sum256(encoded)
		hexSum := hex.EncodeToString(sum[:])
		digests = append(digests, FileDigest{Path: path, SHA256: hexSum})
		fmt.Fprintf(datasetHasher, "%s\x00%s\n", path, hexSum)
	}
	manifest := Manifest{
		Contract: "music-garden.public", ContractVersion: ContractVersion,
		Privacy: "derived-only", Timezone: opts.Timezone,
		PublishedThrough: data.overview.PublishedThrough, SourceRevision: opts.SourceRevision,
		DatasetSHA256: hex.EncodeToString(datasetHasher.Sum(nil)), Counts: data.overview, Files: digests,
	}
	manifestBytes, err := marshalDeterministic(manifest)
	if err != nil {
		return Manifest{}, err
	}
	if err := os.WriteFile(filepath.Join(tmp, "manifest.json"), manifestBytes, 0o644); err != nil {
		return Manifest{}, err
	}
	if err := validateDirectory(tmp, manifest); err != nil {
		return Manifest{}, err
	}
	if err := replaceDirectory(tmp, opts.OutputDir); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func buildData(plays []models.Play, catalog *genres.Catalog, reviewStore *reviews.Store, loc *time.Location) builtData {
	real := make([]models.Play, 0, len(plays))
	for _, play := range plays {
		if isRealPlay(play) {
			real = append(real, play)
		}
	}
	sessions := DetectAlbumSessions(real, catalog, loc)
	sessionsByRelease := map[string][]AlbumSession{}
	for _, session := range sessions {
		sessionsByRelease[session.ReleaseSlug] = append(sessionsByRelease[session.ReleaseSlug], session)
	}

	artistStats := collectArtistStats(real, loc)
	artistOutput := map[string]Artist{}
	for slug, stats := range artistStats {
		record := catalog.Artists.Artists[slug]
		sessionQualified := false
		for releaseSlug := range sessionsByRelease {
			if catalog.Releases.Releases[releaseSlug].PrimaryArtistSlug == slug {
				sessionQualified = true
				break
			}
		}
		intent := (stats.plays >= 3 && len(stats.dates) >= 2) || sessionQualified
		indexed, reasons := applyPublicationDecision(intent && record.SpotifyArtistID != "", "artist", slug, reviewStore, "repeated-intent", "insufficient-listening-evidence")
		artistOutput[slug] = Artist{
			Slug: slug, Name: first(record.Name, stats.name), SortName: record.SortName, ArtistType: record.ArtistType,
			Country: record.Country, LifeSpan: record.LifeSpan, Aliases: record.Aliases,
			SpotifyID: record.SpotifyArtistID, MusicBrainzID: record.MusicBrainzArtistID, WikidataID: record.WikidataID,
			SpotifyURL: record.SpotifyURL, Genres: record.Genres, Summary: record.Summary,
			Attribution: publicAttribution(record.Attribution), Hero: record.Image, Gallery: approvedGallery("artist", slug, record.ImageCandidates, reviewStore),
			SpotifyImages: record.Images, Relationships: record.Relationships,
			TotalPlays: stats.plays, FirstSeen: stats.first, LastSeen: stats.last,
			ListeningMonths: stats.months, TopTracks: topTrackStats(stats.tracks, 12),
			ListenedReleases: releaseSlugsForArtist(slug, sessionsByRelease, catalog),
			Publication:      Publication{Indexed: indexed, Reasons: reasons},
		}
	}

	releaseOutput := buildReleases(real, sessionsByRelease, catalog, reviewStore)
	genreOutput, vectors := buildGenres(real, artistOutput, releaseOutput, catalog, reviewStore, loc)
	attachGenreNeighbors(genreOutput, vectors)
	weeks := buildWeeks(real, artistOutput, loc)
	redirects := buildRedirects(catalog)
	overview := Overview{ArtistCount: len(artistOutput), ListenedReleases: len(releaseOutput), WeeklyPages: len(weeks)}
	for _, artist := range artistOutput {
		if artist.Publication.Indexed {
			overview.IndexedArtists++
		}
	}
	for _, genre := range genreOutput {
		if genre.Publication.Indexed {
			overview.IndexedGenres++
		}
	}
	for _, play := range real {
		if date := publicDate(play.PlayedAt, loc); date > overview.PublishedThrough {
			overview.PublishedThrough = date
		}
	}
	return builtData{artists: artistOutput, genres: genreOutput, releases: releaseOutput, weeks: weeks, redirects: redirects, overview: overview}
}

type artistAccumulator struct {
	name        string
	plays       int
	first, last string
	dates       map[string]bool
	months      map[string]int
	tracks      map[string]*TrackStat
}

func collectArtistStats(plays []models.Play, loc *time.Location) map[string]*artistAccumulator {
	result := map[string]*artistAccumulator{}
	for _, play := range plays {
		if play.ArtistSlug == "" {
			continue
		}
		a := result[play.ArtistSlug]
		if a == nil {
			a = &artistAccumulator{name: play.ArtistName, dates: map[string]bool{}, months: map[string]int{}, tracks: map[string]*TrackStat{}}
			result[play.ArtistSlug] = a
		}
		date := publicDate(play.PlayedAt, loc)
		if date == "" {
			continue
		}
		a.plays++
		a.dates[date] = true
		a.months[date[:7]]++
		if a.first == "" || date < a.first {
			a.first = date
		}
		if date > a.last {
			a.last = date
		}
		key := first(play.TrackID, play.TrackName+"|"+play.AlbumName)
		t := a.tracks[key]
		if t == nil {
			t = &TrackStat{Name: play.TrackName, Artist: play.ArtistName, ArtistSlug: play.ArtistSlug, Album: play.AlbumName, SpotifyURL: play.TrackSpotifyURL, DiscNumber: play.DiscNumber, TrackNumber: play.TrackNumber}
			a.tracks[key] = t
		}
		t.Plays++
	}
	return result
}

func buildReleases(plays []models.Play, sessions map[string][]AlbumSession, catalog *genres.Catalog, reviewStore *reviews.Store) map[string]Release {
	result := map[string]Release{}
	for slug, releaseSessions := range sessions {
		record, ok := catalog.Releases.Releases[slug]
		if !ok || len(releaseSessions) == 0 {
			continue
		}
		sort.Slice(releaseSessions, func(i, j int) bool { return releaseSessions[i].Date < releaseSessions[j].Date })
		trackMap := map[string]*TrackStat{}
		for _, play := range plays {
			if play.ReleaseSlug != slug {
				continue
			}
			key := first(play.TrackID, play.TrackName)
			t := trackMap[key]
			if t == nil {
				t = &TrackStat{Name: play.TrackName, Artist: play.ArtistName, ArtistSlug: play.ArtistSlug, Album: play.AlbumName, SpotifyURL: play.TrackSpotifyURL, DiscNumber: play.DiscNumber, TrackNumber: play.TrackNumber}
				trackMap[key] = t
			}
			t.Plays++
		}
		indexed, reasons := applyPublicationDecision(true, "release", slug, reviewStore, "qualifying-album-session", "no-qualifying-album-session")
		result[slug] = Release{Slug: slug, Name: record.Name, PrimaryArtistSlug: record.PrimaryArtistSlug, PrimaryArtistName: record.PrimaryArtistName, SpotifyAlbumID: record.SpotifyAlbumID, MusicBrainzReleaseGroupID: record.MusicBrainzReleaseGroupID, PrimaryType: record.PrimaryType, SecondaryTypes: record.SecondaryTypes, FirstReleaseDate: record.FirstReleaseDate, ArtistCredits: record.ArtistCredits, Editions: record.Editions, Sessions: releaseSessions, FirstListened: releaseSessions[0].Date, LastListened: releaseSessions[len(releaseSessions)-1].Date, TrackEvidence: topTrackStats(trackMap, 0), Publication: Publication{Indexed: indexed, Reasons: reasons}}
	}
	return result
}

type genreVector map[string]int

func buildGenres(plays []models.Play, artists map[string]Artist, releases map[string]Release, catalog *genres.Catalog, reviewStore *reviews.Store, loc *time.Location) (map[string]Genre, map[string]genreVector) {
	type acc struct {
		plays       int
		artists     map[string]int
		tracks      map[string]*TrackStat
		first, last string
	}
	stats := map[string]*acc{}
	vectors := map[string]genreVector{}
	for _, play := range plays {
		artist, ok := artists[play.ArtistSlug]
		if !ok {
			continue
		}
		date := publicDate(play.PlayedAt, loc)
		for _, slug := range artist.Genres {
			a := stats[slug]
			if a == nil {
				a = &acc{artists: map[string]int{}, tracks: map[string]*TrackStat{}}
				stats[slug] = a
				vectors[slug] = genreVector{}
			}
			a.plays++
			a.artists[play.ArtistSlug]++
			vectors[slug][play.ArtistSlug]++
			if a.first == "" || date < a.first {
				a.first = date
			}
			if date > a.last {
				a.last = date
			}
			key := first(play.TrackID, play.TrackName)
			t := a.tracks[key]
			if t == nil {
				t = &TrackStat{Name: play.TrackName, Artist: play.ArtistName, ArtistSlug: play.ArtistSlug, Album: play.AlbumName, SpotifyURL: play.TrackSpotifyURL}
				a.tracks[key] = t
			}
			t.Plays++
		}
	}
	children := map[string][]string{}
	for _, record := range catalog.Genres.GenreRecords {
		if record.ParentSlug != "" {
			children[record.ParentSlug] = append(children[record.ParentSlug], record.Slug)
		}
	}
	result := map[string]Genre{}
	for slug, a := range stats {
		record := catalog.Genres.GenreRecords[slug]
		eligibleArtists := 0
		topArtists := make([]EntityStat, 0, len(a.artists))
		for artistSlug, count := range a.artists {
			artist := artists[artistSlug]
			if artist.Publication.Indexed {
				eligibleArtists++
			}
			topArtists = append(topArtists, EntityStat{Slug: artistSlug, Name: artist.Name, Plays: count})
		}
		sortEntityStats(topArtists)
		if len(topArtists) > 12 {
			topArtists = topArtists[:12]
		}
		listened := []EntityStat{}
		for releaseSlug, release := range releases {
			artist := artists[release.PrimaryArtistSlug]
			if contains(artist.Genres, slug) {
				listened = append(listened, EntityStat{Slug: releaseSlug, Name: release.Name, Plays: len(release.Sessions)})
			}
		}
		sortEntityStats(listened)
		editorial := record.WorkflowState == genres.WorkflowStatePublishable && strings.TrimSpace(record.Summary) != ""
		indexed, reasons := applyPublicationDecision(a.plays >= 5 && eligibleArtists >= 2 && editorial, "genre", slug, reviewStore, "listening-and-editorial-ready", "insufficient-listening-or-editorial-evidence")
		aliases := []string{}
		for alias, canonical := range catalog.Genres.GenreAliases {
			if canonical == slug {
				aliases = append(aliases, alias)
			}
		}
		sort.Strings(aliases)
		sort.Strings(children[slug])
		result[slug] = Genre{Slug: slug, Name: first(record.DisplayName, humanize(slug)), ParentSlug: record.ParentSlug, ChildSlugs: children[slug], Aliases: aliases, Summary: record.Summary, Attribution: publicAttribution(record.Attribution), Hero: record.Image, PlayCount: a.plays, UniqueArtistCount: len(a.artists), FirstSeen: a.first, LastSeen: a.last, TopArtists: topArtists, TopTracks: topTrackStats(a.tracks, 12), ListenedReleases: listened, Publication: Publication{Indexed: indexed, Reasons: reasons}}
	}
	return result, vectors
}

func attachGenreNeighbors(genresOut map[string]Genre, vectors map[string]genreVector) {
	for slug, genre := range genresOut {
		if !genre.Publication.Indexed {
			continue
		}
		var neighbors []GenreNeighbor
		for otherSlug, other := range genresOut {
			if otherSlug == slug || !other.Publication.Indexed {
				continue
			}
			score, shared := cosine(vectors[slug], vectors[otherSlug])
			if shared >= 2 && score > 0 {
				neighbors = append(neighbors, GenreNeighbor{Slug: otherSlug, Name: other.Name, Score: math.Round(score*1000) / 1000, SharedArtists: shared})
			}
		}
		sort.Slice(neighbors, func(i, j int) bool {
			if neighbors[i].Score != neighbors[j].Score {
				return neighbors[i].Score > neighbors[j].Score
			}
			return neighbors[i].Slug < neighbors[j].Slug
		})
		if len(neighbors) > 6 {
			neighbors = neighbors[:6]
		}
		genre.Neighbors = neighbors
		genresOut[slug] = genre
	}
}

func buildWeeks(plays []models.Play, artists map[string]Artist, loc *time.Location) map[string]Week {
	type acc struct {
		plays                                     int
		duration                                  int
		tracks, artistCounts, albums, genreCounts map[string]int
		trackStats                                map[string]*TrackStat
	}
	groups := map[string]*acc{}
	for _, play := range plays {
		t, err := time.Parse(time.RFC3339Nano, play.PlayedAt)
		if err != nil {
			continue
		}
		year, week := t.In(loc).ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		a := groups[key]
		if a == nil {
			a = &acc{tracks: map[string]int{}, artistCounts: map[string]int{}, albums: map[string]int{}, genreCounts: map[string]int{}, trackStats: map[string]*TrackStat{}}
			groups[key] = a
		}
		a.plays++
		a.duration += play.DurationMS
		a.tracks[first(play.TrackID, play.TrackName)]++
		a.artistCounts[play.ArtistSlug]++
		a.albums[first(play.AlbumID, play.AlbumName)]++
		for _, g := range artists[play.ArtistSlug].Genres {
			a.genreCounts[g]++
		}
		tk := first(play.TrackID, play.TrackName)
		ts := a.trackStats[tk]
		if ts == nil {
			ts = &TrackStat{Name: play.TrackName, Artist: play.ArtistName, ArtistSlug: play.ArtistSlug, Album: play.AlbumName, SpotifyURL: play.TrackSpotifyURL}
			a.trackStats[tk] = ts
		}
		ts.Plays++
	}
	result := map[string]Week{}
	for key, a := range groups {
		parts := strings.Split(key, "-W")
		year := 0
		week := 0
		fmt.Sscanf(parts[0], "%d", &year)
		fmt.Sscanf(parts[1], "%d", &week)
		monday := isoWeekStart(year, week, loc)
		artistList := []EntityStat{}
		for slug, count := range a.artistCounts {
			artistList = append(artistList, EntityStat{Slug: slug, Name: artists[slug].Name, Plays: count})
		}
		sortEntityStats(artistList)
		if len(artistList) > 12 {
			artistList = artistList[:12]
		}
		genreList := []EntityStat{}
		for slug, count := range a.genreCounts {
			genreList = append(genreList, EntityStat{Slug: slug, Name: humanize(slug), Plays: count})
		}
		sortEntityStats(genreList)
		if len(genreList) > 10 {
			genreList = genreList[:10]
		}
		w := Week{WeekStart: monday.Format("2006-01-02"), WeekEnd: monday.AddDate(0, 0, 6).Format("2006-01-02"), TotalPlays: a.plays, UniqueTracks: len(a.tracks), UniqueArtists: len(a.artistCounts), UniqueAlbums: len(a.albums), TotalMinutes: a.duration / 60000, TopArtists: artistList, TopTracks: topTrackStats(a.trackStats, 20), TopGenres: genreList}
		if len(artistList) > 0 {
			w.TopArtistName = artistList[0].Name
			w.TopArtistImages = artists[artistList[0].Slug].SpotifyImages
		}
		result[key] = w
	}
	return result
}

func buildRedirects(catalog *genres.Catalog) Redirects {
	r := Redirects{Artists: map[string]string{}, Genres: map[string]string{}}
	for from, to := range catalog.Artists.ArtistSlugAliases {
		if from != "" && to != "" && from != to {
			r.Artists[from] = to
		}
	}
	for alias, to := range catalog.Genres.GenreAliases {
		from := genres.Slug(alias)
		if from != "" && to != "" && from != to {
			r.Genres[from] = to
		}
	}
	return r
}

func applyPublicationDecision(base bool, kind, slug string, store *reviews.Store, yesReason, noReason string) (bool, []string) {
	indexed := base
	reason := noReason
	if base {
		reason = yesReason
	}
	if d, ok := store.Decision(kind, slug); ok {
		if d.Suppress || d.Status == "rejected" {
			indexed = false
			reason = "suppressed-by-review"
		} else if d.ForcePublish || d.Status == "approved" {
			indexed = true
			reason = "approved-by-review"
		}
	}
	return indexed, []string{reason}
}
func approvedGallery(kind, slug string, candidates []genres.GenreImageRecord, store *reviews.Store) []genres.GenreImageRecord {
	result := []genres.GenreImageRecord{}
	for _, candidate := range candidates {
		key := slug + "#" + first(candidate.FileTitle, candidate.ImageURL)
		if d, ok := store.Decision("media", key); ok && d.Status == "approved" && reusableMediaLicense(candidate.License) {
			result = append(result, candidate)
			if len(result) == 6 {
				break
			}
		}
	}
	return result
}

func reusableMediaLicense(license string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(license))
	return normalized == "PD" || normalized == "PUBLIC DOMAIN" || normalized == "CC0" ||
		strings.HasPrefix(normalized, "CC BY ") || normalized == "CC BY" ||
		strings.HasPrefix(normalized, "CC BY-SA ") || normalized == "CC BY-SA"
}

func publicAttribution(source *genres.GenreSourceAttribution) *genres.GenreSourceAttribution {
	if source == nil {
		return nil
	}
	copy := *source
	if len(copy.RetrievedAt) >= len("2006-01-02") {
		copy.RetrievedAt = copy.RetrievedAt[:len("2006-01-02")]
	}
	return &copy
}
func releaseSlugsForArtist(artist string, sessions map[string][]AlbumSession, catalog *genres.Catalog) []string {
	var result []string
	for slug := range sessions {
		if catalog.Releases.Releases[slug].PrimaryArtistSlug == artist {
			result = append(result, slug)
		}
	}
	sort.Strings(result)
	return result
}
func topTrackStats(m map[string]*TrackStat, limit int) []TrackStat {
	result := make([]TrackStat, 0, len(m))
	for _, v := range m {
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Plays != result[j].Plays {
			return result[i].Plays > result[j].Plays
		}
		if result[i].DiscNumber != result[j].DiscNumber {
			return result[i].DiscNumber < result[j].DiscNumber
		}
		if result[i].TrackNumber != result[j].TrackNumber {
			return result[i].TrackNumber < result[j].TrackNumber
		}
		return result[i].Name < result[j].Name
	})
	if limit > 0 && len(result) > limit {
		return result[:limit]
	}
	return result
}
func sortEntityStats(v []EntityStat) {
	sort.Slice(v, func(i, j int) bool {
		if v[i].Plays != v[j].Plays {
			return v[i].Plays > v[j].Plays
		}
		return v[i].Slug < v[j].Slug
	})
}
func cosine(a, b genreVector) (float64, int) {
	dot, aa, bb, shared := 0, 0, 0, 0
	for k, v := range a {
		aa += v * v
		if w, ok := b[k]; ok {
			dot += v * w
			shared++
		}
	}
	for _, v := range b {
		bb += v * v
	}
	if aa == 0 || bb == 0 {
		return 0, shared
	}
	return float64(dot) / (math.Sqrt(float64(aa)) * math.Sqrt(float64(bb))), shared
}
func publicDate(raw string, loc *time.Location) string {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return t.In(loc).Format("2006-01-02")
}
func isoWeekStart(year, week int, loc *time.Location) time.Time {
	t := time.Date(year, 1, 4, 12, 0, 0, 0, loc)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekOneMonday := t.AddDate(0, 0, 1-weekday)
	return weekOneMonday.AddDate(0, 0, (week-1)*7)
}
func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func humanize(slug string) string {
	parts := strings.Split(slug, "-")
	for i := range parts {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}
func marshalDeterministic(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func validateDirectory(dir string, manifest Manifest) error {
	if !strings.HasPrefix(manifest.ContractVersion, "1.") {
		return fmt.Errorf("unsupported contract version %q", manifest.ContractVersion)
	}
	h := sha256.New()
	for _, file := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file.Path)))
		if err != nil {
			return fmt.Errorf("validate %s: %w", file.Path, err)
		}
		sum := sha256.Sum256(data)
		actual := hex.EncodeToString(sum[:])
		if actual != file.SHA256 {
			return fmt.Errorf("checksum mismatch for %s", file.Path)
		}
		fmt.Fprintf(h, "%s\x00%s\n", file.Path, actual)
	}
	if hex.EncodeToString(h.Sum(nil)) != manifest.DatasetSHA256 {
		return fmt.Errorf("dataset checksum mismatch")
	}
	return nil
}
func replaceDirectory(tmp, target string) error {
	backup := target + ".previous"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("backup previous export: %w", err)
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install export: %w", err)
	}
	_ = os.RemoveAll(backup)
	return nil
}
