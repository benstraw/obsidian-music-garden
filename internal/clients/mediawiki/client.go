package mediawiki

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultWikipediaAPIBase  = "https://en.wikipedia.org/w"
	defaultWikipediaRESTBase = "https://en.wikipedia.org/api/rest_v1"
	defaultCommonsAPIBase    = "https://commons.wikimedia.org/w"
)

// Config controls MediaWiki client behavior.
type Config struct {
	UserAgent         string
	WikipediaAPIBase  string
	WikipediaRESTBase string
	CommonsAPIBase    string
	CacheDir          string
	CacheTTL          time.Duration
	Timeout           time.Duration
}

// Client wraps a small subset of MediaWiki and Wikimedia Commons endpoints.
type Client struct {
	userAgent         string
	wikipediaAPIBase  string
	wikipediaRESTBase string
	commonsAPIBase    string
	cacheDir          string
	cacheTTL          time.Duration
	httpClient        *http.Client
}

type cacheMeta struct {
	Endpoint    string `json:"endpoint"`
	RequestURL  string `json:"request_url"`
	FetchedAt   string `json:"fetched_at"`
	ContentType string `json:"content_type,omitempty"`
}

// SearchResult is one page search candidate.
type SearchResult struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// Summary is the useful subset of a Wikipedia summary response.
type Summary struct {
	Title        string      `json:"title"`
	DisplayTitle string      `json:"displaytitle,omitempty"`
	Extract      string      `json:"extract,omitempty"`
	ContentURLs  ContentURLs `json:"content_urls,omitempty"`
	Thumbnail    *Image      `json:"thumbnail,omitempty"`
	Original     *Image      `json:"originalimage,omitempty"`
	Type         string      `json:"type,omitempty"`
}

// ContentURLs contains canonical page URLs.
type ContentURLs struct {
	Desktop PageURLs `json:"desktop,omitempty"`
	Mobile  PageURLs `json:"mobile,omitempty"`
}

// PageURLs contains a single page URL.
type PageURLs struct {
	Page string `json:"page,omitempty"`
}

// Image is the image payload returned by summary endpoints.
type Image struct {
	Source string `json:"source"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ImageInfo is reduced Commons image metadata.
type ImageInfo struct {
	FileTitle       string `json:"file_title,omitempty"`
	FilePageURL     string `json:"file_page_url,omitempty"`
	ImageURL        string `json:"image_url,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	License         string `json:"license,omitempty"`
	LicenseURL      string `json:"license_url,omitempty"`
	Author          string `json:"author,omitempty"`
	AttributionText string `json:"attribution_text,omitempty"`
}

type pageImagesResponse struct {
	Query struct {
		Pages map[string]struct {
			Images []struct {
				Title string `json:"title"`
			} `json:"images"`
		} `json:"pages"`
	} `json:"query"`
}

type commonsImageInfoResponse struct {
	Query struct {
		Pages map[string]struct {
			Title     string `json:"title"`
			ImageInfo []struct {
				URL         string `json:"url"`
				Description string `json:"descriptionurl"`
				ThumbURL    string `json:"thumburl"`
				Width       int    `json:"width"`
				Height      int    `json:"height"`
				ExtMetadata struct {
					LicenseShortName struct {
						Value string `json:"value"`
					} `json:"LicenseShortName"`
					LicenseURL struct {
						Value string `json:"value"`
					} `json:"LicenseUrl"`
					Artist struct {
						Value string `json:"value"`
					} `json:"Artist"`
					Attribution struct {
						Value string `json:"value"`
					} `json:"Attribution"`
				} `json:"extmetadata"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

// FetchResult returns value plus raw payload metadata.
type FetchResult[T any] struct {
	Value      T
	Body       []byte
	Endpoint   string
	RequestURL string
	FetchedAt  time.Time
	FromCache  bool
}

// NewClient creates a new MediaWiki client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.UserAgent) == "" {
		return nil, fmt.Errorf("mediawiki user agent is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 24 * time.Hour
	}
	if cfg.WikipediaAPIBase == "" {
		cfg.WikipediaAPIBase = defaultWikipediaAPIBase
	}
	if cfg.WikipediaRESTBase == "" {
		cfg.WikipediaRESTBase = defaultWikipediaRESTBase
	}
	if cfg.CommonsAPIBase == "" {
		cfg.CommonsAPIBase = defaultCommonsAPIBase
	}
	return &Client{
		userAgent:         cfg.UserAgent,
		wikipediaAPIBase:  strings.TrimRight(cfg.WikipediaAPIBase, "/"),
		wikipediaRESTBase: strings.TrimRight(cfg.WikipediaRESTBase, "/"),
		commonsAPIBase:    strings.TrimRight(cfg.CommonsAPIBase, "/"),
		cacheDir:          strings.TrimSpace(cfg.CacheDir),
		cacheTTL:          cfg.CacheTTL,
		httpClient:        &http.Client{Timeout: cfg.Timeout},
	}, nil
}

// SearchPages searches English Wikipedia page titles for a genre query.
func (c *Client) SearchPages(query string) (FetchResult[[]SearchResult], error) {
	params := url.Values{}
	params.Set("action", "opensearch")
	params.Set("search", query)
	params.Set("limit", "10")
	params.Set("namespace", "0")
	params.Set("format", "json")
	endpoint := "/api.php"
	requestURL := c.wikipediaAPIBase + endpoint + "?" + params.Encode()
	cacheKey := filepath.Join("search", slugForCache(query))
	body, meta, fromCache, err := c.fetchBytes(requestURL, endpoint, cacheKey)
	if err != nil {
		return FetchResult[[]SearchResult]{}, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return FetchResult[[]SearchResult]{}, fmt.Errorf("decode wikipedia search: %w", err)
	}
	var titles, descriptions, urls []string
	if len(raw) >= 4 {
		_ = json.Unmarshal(raw[1], &titles)
		_ = json.Unmarshal(raw[2], &descriptions)
		_ = json.Unmarshal(raw[3], &urls)
	}
	results := make([]SearchResult, 0, len(titles))
	for i, title := range titles {
		result := SearchResult{Title: title}
		if i < len(descriptions) {
			result.Description = descriptions[i]
		}
		if i < len(urls) {
			result.URL = urls[i]
		}
		results = append(results, result)
	}
	return FetchResult[[]SearchResult]{
		Value:      results,
		Body:       body,
		Endpoint:   meta.Endpoint,
		RequestURL: meta.RequestURL,
		FetchedAt:  meta.fetchedAt(),
		FromCache:  fromCache,
	}, nil
}

// GetSummary fetches a page summary from English Wikipedia.
func (c *Client) GetSummary(title string) (FetchResult[Summary], error) {
	endpoint := "/page/summary/" + url.PathEscape(title)
	requestURL := c.wikipediaRESTBase + endpoint
	cacheKey := filepath.Join("summaries", slugForCache(title))
	body, meta, fromCache, err := c.fetchBytes(requestURL, endpoint, cacheKey)
	if err != nil {
		return FetchResult[Summary]{}, err
	}
	var summary Summary
	if err := json.Unmarshal(body, &summary); err != nil {
		return FetchResult[Summary]{}, fmt.Errorf("decode wikipedia summary: %w", err)
	}
	return FetchResult[Summary]{
		Value:      summary,
		Body:       body,
		Endpoint:   meta.Endpoint,
		RequestURL: meta.RequestURL,
		FetchedAt:  meta.fetchedAt(),
		FromCache:  fromCache,
	}, nil
}

// GetPageImages lists image titles associated with a Wikipedia page.
func (c *Client) GetPageImages(title string) (FetchResult[[]string], error) {
	params := url.Values{}
	params.Set("action", "query")
	params.Set("prop", "images")
	params.Set("titles", title)
	params.Set("imlimit", "10")
	params.Set("format", "json")
	endpoint := "/api.php"
	requestURL := c.wikipediaAPIBase + endpoint + "?" + params.Encode()
	cacheKey := filepath.Join("page-images", slugForCache(title))
	body, meta, fromCache, err := c.fetchBytes(requestURL, endpoint, cacheKey)
	if err != nil {
		return FetchResult[[]string]{}, err
	}
	var response pageImagesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return FetchResult[[]string]{}, fmt.Errorf("decode wikipedia page images: %w", err)
	}
	var images []string
	for _, page := range response.Query.Pages {
		for _, image := range page.Images {
			images = append(images, image.Title)
		}
	}
	return FetchResult[[]string]{
		Value:      images,
		Body:       body,
		Endpoint:   meta.Endpoint,
		RequestURL: meta.RequestURL,
		FetchedAt:  meta.fetchedAt(),
		FromCache:  fromCache,
	}, nil
}

// GetCommonsImageInfo fetches Commons file metadata for one File: title.
func (c *Client) GetCommonsImageInfo(fileTitle string) (FetchResult[ImageInfo], error) {
	params := url.Values{}
	params.Set("action", "query")
	params.Set("prop", "imageinfo")
	params.Set("titles", fileTitle)
	params.Set("iiprop", "url|extmetadata|dimensions")
	params.Set("iiurlwidth", "640")
	params.Set("format", "json")
	endpoint := "/api.php"
	requestURL := c.commonsAPIBase + endpoint + "?" + params.Encode()
	cacheKey := filepath.Join("commons-images", slugForCache(fileTitle))
	body, meta, fromCache, err := c.fetchBytes(requestURL, endpoint, cacheKey)
	if err != nil {
		return FetchResult[ImageInfo]{}, err
	}
	var response commonsImageInfoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return FetchResult[ImageInfo]{}, fmt.Errorf("decode commons image info: %w", err)
	}
	var info ImageInfo
	for _, page := range response.Query.Pages {
		info.FileTitle = page.Title
		if len(page.ImageInfo) == 0 {
			break
		}
		imageInfo := page.ImageInfo[0]
		info.FilePageURL = imageInfo.Description
		info.ImageURL = imageInfo.URL
		info.ThumbnailURL = imageInfo.ThumbURL
		info.Width = imageInfo.Width
		info.Height = imageInfo.Height
		info.License = imageInfo.ExtMetadata.LicenseShortName.Value
		info.LicenseURL = imageInfo.ExtMetadata.LicenseURL.Value
		info.Author = imageInfo.ExtMetadata.Artist.Value
		info.AttributionText = imageInfo.ExtMetadata.Attribution.Value
		break
	}
	return FetchResult[ImageInfo]{
		Value:      info,
		Body:       body,
		Endpoint:   meta.Endpoint,
		RequestURL: meta.RequestURL,
		FetchedAt:  meta.fetchedAt(),
		FromCache:  fromCache,
	}, nil
}

func (c *Client) fetchBytes(requestURL, endpoint, cacheKey string) ([]byte, cacheMeta, bool, error) {
	if body, meta, ok := c.readCache(cacheKey); ok {
		return body, meta, true, nil
	}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, cacheMeta{}, false, fmt.Errorf("build mediawiki request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, cacheMeta{}, false, fmt.Errorf("mediawiki request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, cacheMeta{}, false, fmt.Errorf("read mediawiki response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, cacheMeta{}, false, fmt.Errorf("mediawiki returned %d for %s", resp.StatusCode, requestURL)
	}
	meta := cacheMeta{
		Endpoint:    endpoint,
		RequestURL:  requestURL,
		FetchedAt:   time.Now().UTC().Format(time.RFC3339),
		ContentType: resp.Header.Get("Content-Type"),
	}
	c.writeCache(cacheKey, body, meta)
	return body, meta, false, nil
}

func (c *Client) readCache(cacheKey string) ([]byte, cacheMeta, bool) {
	if c.cacheDir == "" || cacheKey == "" {
		return nil, cacheMeta{}, false
	}
	bodyPath := filepath.Join(c.cacheDir, cacheKey+".json")
	metaPath := filepath.Join(c.cacheDir, cacheKey+".manifest.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, cacheMeta{}, false
	}
	var meta cacheMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, cacheMeta{}, false
	}
	if time.Since(meta.fetchedAt()) > c.cacheTTL {
		return nil, cacheMeta{}, false
	}
	body, err := os.ReadFile(bodyPath)
	if err != nil {
		return nil, cacheMeta{}, false
	}
	return body, meta, true
}

func (c *Client) writeCache(cacheKey string, body []byte, meta cacheMeta) {
	if c.cacheDir == "" || cacheKey == "" {
		return
	}
	bodyPath := filepath.Join(c.cacheDir, cacheKey+".json")
	metaPath := filepath.Join(c.cacheDir, cacheKey+".manifest.json")
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

func (m cacheMeta) fetchedAt() time.Time {
	t, err := time.Parse(time.RFC3339, m.FetchedAt)
	if err != nil {
		return time.Time{}
	}
	return t
}

func slugForCache(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, ":", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	value = strings.Trim(value, "-")
	if value == "" {
		return "lookup"
	}
	return value
}
