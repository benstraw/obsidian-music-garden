package musicbrainz

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benstraw/music-garden/internal/genres"
)

const defaultBaseURL = "https://musicbrainz.org/ws/2"

// ErrNoResults is returned when a search succeeds but returns no matches.
var ErrNoResults = errors.New("no MusicBrainz results")

// Config controls MusicBrainz client behavior.
type Config struct {
	UserAgent string
	BaseURL   string
	CacheDir  string
	CacheTTL  time.Duration
	Timeout   time.Duration
}

// Client wraps the MusicBrainz web service with rate limiting and disk cache.
type Client struct {
	baseURL    string
	userAgent  string
	cacheDir   string
	cacheTTL   time.Duration
	httpClient *http.Client

	mu          sync.Mutex
	lastRequest time.Time
}

type fetchMeta struct {
	Endpoint    string `json:"endpoint"`
	RequestURL  string `json:"request_url"`
	FetchedAt   string `json:"fetched_at"`
	ContentType string `json:"content_type,omitempty"`
}

// Score decodes MusicBrainz search scores that may arrive as either JSON
// numbers or strings depending on endpoint and deployment behavior.
type Score int

// UnmarshalJSON accepts either a quoted or unquoted integer.
func (s *Score) UnmarshalJSON(data []byte) error {
	var asInt int
	if err := json.Unmarshal(data, &asInt); err == nil {
		*s = Score(asInt)
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err != nil {
		return err
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(asString))
	if err != nil {
		return err
	}
	*s = Score(parsed)
	return nil
}

// SearchArtistResult is a lightweight MusicBrainz artist match.
type SearchArtistResult struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	SortName       string     `json:"sort-name"`
	Disambiguation string     `json:"disambiguation,omitempty"`
	Genres         []TagCount `json:"genres,omitempty"`
	Tags           []TagCount `json:"tags,omitempty"`
	Score          Score      `json:"score,omitempty"`
}

// Artist is a MusicBrainz artist lookup result.
type Artist struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	SortName       string     `json:"sort-name"`
	Disambiguation string     `json:"disambiguation,omitempty"`
	Country        string     `json:"country,omitempty"`
	Type           string     `json:"type,omitempty"`
	Genres         []TagCount `json:"genres,omitempty"`
	Tags           []TagCount `json:"tags,omitempty"`
	LifeSpan       LifeSpan   `json:"life-span,omitempty"`
	Aliases        []Alias    `json:"aliases,omitempty"`
	Relations      []Relation `json:"relations,omitempty"`
}

type LifeSpan struct {
	Begin string `json:"begin,omitempty"`
	End   string `json:"end,omitempty"`
	Ended bool   `json:"ended,omitempty"`
}

type Alias struct {
	Name string `json:"name"`
}

type URLRef struct {
	Resource string `json:"resource"`
}

type Relation struct {
	Type      string        `json:"type"`
	Direction string        `json:"direction,omitempty"`
	Artist    *CreditArtist `json:"artist,omitempty"`
	URL       *URLRef       `json:"url,omitempty"`
	Begin     string        `json:"begin,omitempty"`
	End       string        `json:"end,omitempty"`
}

// SearchReleaseGroupResult is a lightweight MusicBrainz release-group match.
type SearchReleaseGroupResult struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	PrimaryType      string         `json:"primary-type,omitempty"`
	FirstReleaseDate string         `json:"first-release-date,omitempty"`
	ArtistCredit     []ArtistCredit `json:"artist-credit,omitempty"`
	Genres           []TagCount     `json:"genres,omitempty"`
	Tags             []TagCount     `json:"tags,omitempty"`
	Score            Score          `json:"score,omitempty"`
}

// ReleaseGroup is a MusicBrainz release-group lookup result.
type ReleaseGroup struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	PrimaryType      string         `json:"primary-type,omitempty"`
	SecondaryTypes   []string       `json:"secondary-types,omitempty"`
	FirstReleaseDate string         `json:"first-release-date,omitempty"`
	ArtistCredit     []ArtistCredit `json:"artist-credit,omitempty"`
	Genres           []TagCount     `json:"genres,omitempty"`
	Tags             []TagCount     `json:"tags,omitempty"`
	Releases         []ReleaseRef   `json:"releases,omitempty"`
}

// Release is a lightweight MusicBrainz release match.
type Release struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Date         string         `json:"date,omitempty"`
	ArtistCredit []ArtistCredit `json:"artist-credit,omitempty"`
	ReleaseGroup *ReleaseRef    `json:"release-group,omitempty"`
	Score        Score          `json:"score,omitempty"`
}

// ReleaseRef represents a related release or release-group identifier.
type ReleaseRef struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// ArtistCredit is a reduced MusicBrainz artist credit item.
type ArtistCredit struct {
	Name   string       `json:"name,omitempty"`
	Artist CreditArtist `json:"artist"`
}

// CreditArtist is the nested artist object inside artist credits.
type CreditArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TagCount represents MusicBrainz genres or tags.
type TagCount struct {
	Name  string `json:"name"`
	Count int    `json:"count,omitempty"`
}

type artistSearchResponse struct {
	Artists []SearchArtistResult `json:"artists"`
}

type releaseGroupSearchResponse struct {
	ReleaseGroups []SearchReleaseGroupResult `json:"release-groups"`
}

type releaseSearchResponse struct {
	Releases []Release `json:"releases"`
}

// FetchResult exposes the raw payload and request metadata used by the caller.
type FetchResult[T any] struct {
	Value      T
	Body       []byte
	Endpoint   string
	RequestURL string
	FetchedAt  time.Time
	FromCache  bool
}

// NewClient creates a new MusicBrainz client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.UserAgent) == "" {
		return nil, fmt.Errorf("musicbrainz user agent is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL:    baseURL,
		userAgent:  cfg.UserAgent,
		cacheDir:   strings.TrimSpace(cfg.CacheDir),
		cacheTTL:   cacheTTL,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// SearchArtistByName searches artists by display name.
func (c *Client) SearchArtistByName(name string) (FetchResult[[]SearchArtistResult], error) {
	query := fmt.Sprintf(`artist:"%s"`, escapeQuery(name))
	endpoint := "/artist"
	params := url.Values{}
	params.Set("query", query)
	params.Set("limit", "5")
	params.Set("fmt", "json")
	stem := genres.Slug(name)
	if stem == "" {
		stem = "artist-search"
	}
	var resp artistSearchResponse
	result, err := getJSON(c, endpoint, params, filepath.Join("artist-search", stem), &resp)
	if err != nil {
		return FetchResult[[]SearchArtistResult]{}, err
	}
	if len(resp.Artists) == 0 {
		return FetchResult[[]SearchArtistResult]{}, ErrNoResults
	}
	return FetchResult[[]SearchArtistResult]{
		Value:      resp.Artists,
		Body:       result.Body,
		Endpoint:   result.Endpoint,
		RequestURL: result.RequestURL,
		FetchedAt:  result.FetchedAt,
		FromCache:  result.FromCache,
	}, nil
}

// GetArtistByID fetches one MusicBrainz artist with genres and tags.
func (c *Client) GetArtistByID(id string) (FetchResult[Artist], error) {
	endpoint := "/artist/" + url.PathEscape(id)
	params := url.Values{}
	params.Set("inc", "genres+tags+aliases+artist-rels+url-rels")
	params.Set("fmt", "json")
	var artist Artist
	return getJSON(c, endpoint, params, filepath.Join("artists", id), &artist)
}

// SearchReleaseGroups finds likely release-group matches for an artist/title pair.
func (c *Client) SearchReleaseGroups(artistName, releaseName string) (FetchResult[[]SearchReleaseGroupResult], error) {
	query := fmt.Sprintf(`releasegroup:"%s" AND artist:"%s"`, escapeQuery(releaseName), escapeQuery(artistName))
	endpoint := "/release-group"
	params := url.Values{}
	params.Set("query", query)
	params.Set("limit", "5")
	params.Set("fmt", "json")
	stem := joinedStem(artistName, releaseName)
	var resp releaseGroupSearchResponse
	result, err := getJSON(c, endpoint, params, filepath.Join("release-group-search", stem), &resp)
	if err != nil {
		return FetchResult[[]SearchReleaseGroupResult]{}, err
	}
	if len(resp.ReleaseGroups) == 0 {
		return FetchResult[[]SearchReleaseGroupResult]{}, ErrNoResults
	}
	return FetchResult[[]SearchReleaseGroupResult]{
		Value:      resp.ReleaseGroups,
		Body:       result.Body,
		Endpoint:   result.Endpoint,
		RequestURL: result.RequestURL,
		FetchedAt:  result.FetchedAt,
		FromCache:  result.FromCache,
	}, nil
}

// SearchReleases finds release matches for an artist/title pair.
func (c *Client) SearchReleases(artistName, releaseName string) (FetchResult[[]Release], error) {
	query := fmt.Sprintf(`release:"%s" AND artist:"%s"`, escapeQuery(releaseName), escapeQuery(artistName))
	endpoint := "/release"
	params := url.Values{}
	params.Set("query", query)
	params.Set("limit", "5")
	params.Set("fmt", "json")
	stem := joinedStem(artistName, releaseName)
	var resp releaseSearchResponse
	result, err := getJSON(c, endpoint, params, filepath.Join("releases", stem), &resp)
	if err != nil {
		return FetchResult[[]Release]{}, err
	}
	if len(resp.Releases) == 0 {
		return FetchResult[[]Release]{}, ErrNoResults
	}
	return FetchResult[[]Release]{
		Value:      resp.Releases,
		Body:       result.Body,
		Endpoint:   result.Endpoint,
		RequestURL: result.RequestURL,
		FetchedAt:  result.FetchedAt,
		FromCache:  result.FromCache,
	}, nil
}

// GetReleaseGroupByID fetches one MusicBrainz release-group with genres and tags.
func (c *Client) GetReleaseGroupByID(id string) (FetchResult[ReleaseGroup], error) {
	endpoint := "/release-group/" + url.PathEscape(id)
	params := url.Values{}
	params.Set("inc", "artists+genres+tags+releases")
	params.Set("fmt", "json")
	var group ReleaseGroup
	return getJSON(c, endpoint, params, filepath.Join("release-groups", id), &group)
}

func getJSON[T any](c *Client, endpoint string, params url.Values, cacheKey string, dst *T) (FetchResult[T], error) {
	requestURL := c.baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	if body, meta, ok := c.readCache(cacheKey); ok {
		if err := json.Unmarshal(body, dst); err != nil {
			return FetchResult[T]{}, fmt.Errorf("decode cached MusicBrainz response: %w", err)
		}
		fetchedAt, _ := time.Parse(time.RFC3339, meta.FetchedAt)
		return FetchResult[T]{
			Value:      *dst,
			Body:       body,
			Endpoint:   meta.Endpoint,
			RequestURL: meta.RequestURL,
			FetchedAt:  fetchedAt,
			FromCache:  true,
		}, nil
	}

	body, contentType, fetchedAt, err := c.doRequest(requestURL)
	if err != nil {
		return FetchResult[T]{}, err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return FetchResult[T]{}, fmt.Errorf("decode MusicBrainz response: %w", err)
	}

	c.writeCache(cacheKey, body, fetchMeta{
		Endpoint:    endpoint,
		RequestURL:  requestURL,
		FetchedAt:   fetchedAt.Format(time.RFC3339),
		ContentType: contentType,
	})

	return FetchResult[T]{
		Value:      *dst,
		Body:       body,
		Endpoint:   endpoint,
		RequestURL: requestURL,
		FetchedAt:  fetchedAt,
	}, nil
}

func (c *Client) doRequest(requestURL string) ([]byte, string, time.Time, error) {
	backoff := time.Second
	for attempt := 0; attempt < 4; attempt++ {
		if err := c.waitForRateLimit(); err != nil {
			return nil, "", time.Time{}, err
		}

		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, "", time.Time{}, fmt.Errorf("build MusicBrainz request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, "", time.Time{}, fmt.Errorf("MusicBrainz request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, "", time.Time{}, fmt.Errorf("read MusicBrainz response: %w", readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, "", time.Time{}, ErrNoResults
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, "", time.Time{}, fmt.Errorf("MusicBrainz returned %d for %s", resp.StatusCode, requestURL)
		}
		return body, resp.Header.Get("Content-Type"), time.Now().UTC(), nil
	}
	return nil, "", time.Time{}, fmt.Errorf("MusicBrainz request exhausted retries for %s", requestURL)
}

func (c *Client) waitForRateLimit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastRequest.IsZero() {
		c.lastRequest = time.Now()
		return nil
	}
	wait := time.Second - time.Since(c.lastRequest)
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		<-timer.C
	}
	c.lastRequest = time.Now()
	return nil
}

func (c *Client) readCache(cacheKey string) ([]byte, fetchMeta, bool) {
	if c.cacheDir == "" || cacheKey == "" {
		return nil, fetchMeta{}, false
	}
	bodyPath, metaPath := c.cachePaths(cacheKey)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fetchMeta{}, false
	}
	var meta fetchMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fetchMeta{}, false
	}
	fetchedAt, err := time.Parse(time.RFC3339, meta.FetchedAt)
	if err != nil || time.Since(fetchedAt) > c.cacheTTL {
		return nil, fetchMeta{}, false
	}
	body, err := os.ReadFile(bodyPath)
	if err != nil {
		return nil, fetchMeta{}, false
	}
	return body, meta, true
}

func (c *Client) writeCache(cacheKey string, body []byte, meta fetchMeta) {
	if c.cacheDir == "" || cacheKey == "" {
		return
	}
	bodyPath, metaPath := c.cachePaths(cacheKey)
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0755); err != nil {
		return
	}
	if err := os.WriteFile(bodyPath, body, 0644); err != nil {
		return
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(metaPath, metaBytes, 0644)
}

func (c *Client) cachePaths(cacheKey string) (string, string) {
	bodyPath := filepath.Join(c.cacheDir, cacheKey+".json")
	metaPath := filepath.Join(c.cacheDir, cacheKey+".manifest.json")
	return bodyPath, metaPath
}

func escapeQuery(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func joinedStem(parts ...string) string {
	slugs := make([]string, 0, len(parts))
	for _, part := range parts {
		slug := genres.Slug(part)
		if slug == "" {
			continue
		}
		slugs = append(slugs, slug)
	}
	if len(slugs) == 0 {
		return "lookup"
	}
	return strings.Join(slugs, "--")
}

// GenreNames returns deduplicated MusicBrainz genre/tag names.
func GenreNames(genresList, tags []TagCount) []string {
	seen := map[string]bool{}
	var names []string
	for _, entry := range append(append([]TagCount(nil), genresList...), tags...) {
		name := strings.TrimSpace(entry.Name)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, name)
	}
	return names
}

// PrimaryArtistName returns the first credited artist name.
func PrimaryArtistName(credits []ArtistCredit) string {
	if len(credits) == 0 {
		return ""
	}
	if credits[0].Artist.Name != "" {
		return credits[0].Artist.Name
	}
	return credits[0].Name
}
