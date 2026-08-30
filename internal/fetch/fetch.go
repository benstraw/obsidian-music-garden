package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/benstraw/music-garden/internal/client"
	"github.com/benstraw/music-garden/internal/models"
)

// --- Spotify API response types ---

type recentlyPlayedResponse struct {
	Items []recentlyPlayedItem `json:"items"`
}

type recentlyPlayedItem struct {
	Track    *spotifyTrack `json:"track"` // nil for podcasts
	PlayedAt string        `json:"played_at"`
}

type spotifyTrack struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Artists      []spotifyArtist   `json:"artists"`
	Album        spotifyAlbum      `json:"album"`
	DiscNumber   int               `json:"disc_number"`
	TrackNumber  int               `json:"track_number"`
	DurationMS   int               `json:"duration_ms"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type spotifyArtist struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Genres       []string          `json:"genres"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type spotifyAlbum struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	AlbumType    string            `json:"album_type"`
	ReleaseDate  string            `json:"release_date"`
	TotalTracks  int               `json:"total_tracks"`
	Artists      []spotifyArtist   `json:"artists"`
	Images       []spotifyImage    `json:"images"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type spotifyAlbumFull struct {
	spotifyAlbum
	Tracks spotifyAlbumTracks `json:"tracks"`
}

type spotifyAlbumTracks struct {
	Items []spotifyTrack `json:"items"`
}

type topTracksResponse struct {
	Items []topTrackItem `json:"items"`
}

type topTrackItem struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Artists []spotifyArtist `json:"artists"`
}

type topArtistsResponse struct {
	Items []topArtistItem `json:"items"`
}

type spotifyImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type topArtistItem struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Genres       []string          `json:"genres"`
	ExternalURLs map[string]string `json:"external_urls"`
	Images       []spotifyImage    `json:"images"`
}

func toModelImages(imgs []spotifyImage) []models.ArtistImage {
	result := make([]models.ArtistImage, 0, len(imgs))
	for _, img := range imgs {
		result = append(result, models.ArtistImage{
			URL:    img.URL,
			Height: img.Height,
			Width:  img.Width,
		})
	}
	return result
}

// GetRecentlyPlayed fetches up to 50 recently played tracks.
// Podcast episodes (items with no track key) are filtered silently.
func GetRecentlyPlayed(c *client.Client) ([]models.Play, error) {
	plays, _, err := GetRecentlyPlayedRaw(c)
	return plays, err
}

// GetRecentlyPlayedRaw fetches up to 50 recently played tracks and returns the
// unchanged Spotify response body alongside the mapped plays.
func GetRecentlyPlayedRaw(c *client.Client) ([]models.Play, []byte, error) {
	params := url.Values{}
	params.Set("limit", "50")

	body, err := c.Get("/me/player/recently-played", params)
	if err != nil {
		return nil, nil, fmt.Errorf("recently-played: %w", err)
	}

	var resp recentlyPlayedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("recently-played decode: %w", err)
	}

	var plays []models.Play
	for _, item := range resp.Items {
		if item.Track == nil {
			continue // podcast episode — skip silently
		}
		p := itemToPlay(item)
		plays = append(plays, p)
	}
	return plays, body, nil
}

// itemToPlay maps a recently-played API item to a Play struct (primary artist only).
func itemToPlay(item recentlyPlayedItem) models.Play {
	t := trackDetailsFromSpotify(item.Track)
	var artistID, artistName, artistURL string
	if len(t.Artists) > 0 {
		artistID = t.Artists[0].ID
		artistName = t.Artists[0].Name
		artistURL = t.Artists[0].SpotifyURL
	}
	additionalArtists := []models.PlayArtist(nil)
	if len(t.Artists) > 1 {
		additionalArtists = append(additionalArtists, t.Artists[1:]...)
	}
	return models.Play{
		PlayedAt:          item.PlayedAt,
		Source:            "spotify",
		TrackID:           t.ID,
		TrackName:         t.Name,
		ArtistID:          artistID,
		ArtistName:        artistName,
		ArtistSpotifyURL:  artistURL,
		AdditionalArtists: additionalArtists,
		AlbumArtists:      t.AlbumArtists,
		AlbumID:           t.AlbumID,
		AlbumName:         t.AlbumName,
		AlbumType:         t.AlbumType,
		AlbumReleaseDate:  t.AlbumReleaseDate,
		AlbumTotalTracks:  t.AlbumTotalTracks,
		AlbumImages:       t.AlbumImages,
		DiscNumber:        t.DiscNumber,
		TrackNumber:       t.TrackNumber,
		DurationMS:        t.DurationMS,
		TrackSpotifyURL:   t.TrackSpotifyURL,
	}
}

// GetTopTracks fetches the user's top 50 tracks for the given time range.
// timeRange: "short_term" | "medium_term" | "long_term"
func GetTopTracks(c *client.Client, timeRange string) ([]models.TopTrack, error) {
	params := url.Values{}
	params.Set("limit", "50")
	params.Set("time_range", timeRange)

	body, err := c.Get("/me/top/tracks", params)
	if err != nil {
		return nil, fmt.Errorf("top/tracks: %w", err)
	}

	var resp topTracksResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("top/tracks decode: %w", err)
	}

	tracks := make([]models.TopTrack, 0, len(resp.Items))
	for _, item := range resp.Items {
		var artistName string
		if len(item.Artists) > 0 {
			artistName = item.Artists[0].Name
		}
		tracks = append(tracks, models.TopTrack{
			ID:         item.ID,
			Name:       item.Name,
			ArtistName: artistName,
		})
	}
	return tracks, nil
}

// --- Batch artist lookup ---

type artistsResponse struct {
	Artists []topArtistItem `json:"artists"`
}

type tracksResponse struct {
	Tracks []*spotifyTrack `json:"tracks"`
}

type albumsResponse struct {
	Albums []*spotifyAlbumFull `json:"albums"`
}

// GetArtists fetches artist details for up to 50 IDs in a single request.
func GetArtists(c *client.Client, ids []string) ([]models.TopArtist, error) {
	artists, _, err := GetArtistsRaw(c, ids)
	return artists, err
}

// GetArtistsRaw fetches artist details for up to 50 IDs in a single request
// and returns the unchanged Spotify response body.
func GetArtistsRaw(c *client.Client, ids []string) ([]models.TopArtist, []byte, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	if len(ids) > 50 {
		return nil, nil, fmt.Errorf("GetArtists: max 50 IDs per request, got %d", len(ids))
	}

	params := url.Values{}
	params.Set("ids", joinIDs(ids))

	body, err := c.Get("/artists", params)
	if err != nil {
		return nil, nil, fmt.Errorf("artists: %w", err)
	}

	var resp artistsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("artists decode: %w", err)
	}

	artists := make([]models.TopArtist, 0, len(resp.Artists))
	for _, item := range resp.Artists {
		artists = append(artists, models.TopArtist{
			ID:         item.ID,
			Name:       item.Name,
			Genres:     item.Genres,
			SpotifyURL: item.ExternalURLs["spotify"],
			Images:     toModelImages(item.Images),
		})
	}
	return artists, body, nil
}

// GetArtistsBatch fetches artist details for any number of IDs, chunking into batches of 50.
func GetArtistsBatch(c *client.Client, ids []string) ([]models.TopArtist, error) {
	artists, _, err := GetArtistsBatchRaw(c, ids)
	return artists, err
}

// GetArtistsBatchRaw fetches artist details for any number of IDs, chunking
// into batches of 50, and returns the unchanged response body for each batch.
func GetArtistsBatchRaw(c *client.Client, ids []string) ([]models.TopArtist, [][]byte, error) {
	var all []models.TopArtist
	var bodies [][]byte
	for i := 0; i < len(ids); i += 50 {
		end := min(i+50, len(ids))
		batch, body, err := GetArtistsRaw(c, ids[i:end])
		if err != nil {
			return nil, nil, err
		}
		all = append(all, batch...)
		if len(body) > 0 {
			bodies = append(bodies, body)
		}
	}
	return all, bodies, nil
}

// joinIDs joins IDs with commas.
func joinIDs(ids []string) string {
	return strings.Join(ids, ",")
}

// GetTracks fetches full Spotify track objects for up to 50 IDs in one request.
func GetTracks(c *client.Client, ids []string) ([]models.TrackDetails, error) {
	tracks, _, err := GetTracksRaw(c, ids)
	return tracks, err
}

// GetTracksRaw fetches full Spotify track objects for up to 50 IDs in one request
// and returns the unchanged Spotify response body.
func GetTracksRaw(c *client.Client, ids []string) ([]models.TrackDetails, []byte, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	if len(ids) > 50 {
		return nil, nil, fmt.Errorf("GetTracks: max 50 IDs per request, got %d", len(ids))
	}

	params := url.Values{}
	params.Set("ids", joinIDs(ids))

	body, err := c.Get("/tracks", params)
	if err != nil {
		return nil, nil, fmt.Errorf("tracks: %w", err)
	}

	var resp tracksResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("tracks decode: %w", err)
	}
	tracks := make([]models.TrackDetails, 0, len(resp.Tracks))
	for _, track := range resp.Tracks {
		if track == nil {
			continue
		}
		tracks = append(tracks, trackDetailsFromSpotify(track))
	}
	return tracks, body, nil
}

// GetTracksBatch fetches full Spotify track objects for any number of IDs, chunking into batches of 50.
func GetTracksBatch(c *client.Client, ids []string) ([]models.TrackDetails, error) {
	tracks, _, err := GetTracksBatchRaw(c, ids)
	return tracks, err
}

// GetTracksBatchRaw fetches full Spotify track objects for any number of IDs, chunking
// into batches of 50, and returns the unchanged response body for each batch.
func GetTracksBatchRaw(c *client.Client, ids []string) ([]models.TrackDetails, [][]byte, error) {
	var all []models.TrackDetails
	var bodies [][]byte
	for i := 0; i < len(ids); i += 50 {
		end := min(i+50, len(ids))
		batch, body, err := GetTracksRaw(c, ids[i:end])
		if err != nil {
			return nil, nil, err
		}
		all = append(all, batch...)
		if len(body) > 0 {
			bodies = append(bodies, body)
		}
	}
	return all, bodies, nil
}

// GetAlbumsRaw fetches complete album editions for up to 20 Spotify IDs.
func GetAlbumsRaw(c *client.Client, ids []string) ([]models.AlbumDetails, []byte, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	if len(ids) > 20 {
		return nil, nil, fmt.Errorf("GetAlbums: max 20 IDs per request, got %d", len(ids))
	}
	params := url.Values{}
	params.Set("ids", joinIDs(ids))
	body, err := c.Get("/albums", params)
	if err != nil {
		return nil, nil, fmt.Errorf("albums: %w", err)
	}
	var resp albumsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("albums decode: %w", err)
	}
	albums := make([]models.AlbumDetails, 0, len(resp.Albums))
	for _, album := range resp.Albums {
		if album == nil {
			continue
		}
		artists := make([]models.PlayArtist, 0, len(album.Artists))
		for _, artist := range album.Artists {
			artists = append(artists, models.PlayArtist{ID: artist.ID, Name: artist.Name, SpotifyURL: artist.ExternalURLs["spotify"]})
		}
		details := models.AlbumDetails{
			ID: album.ID, Name: album.Name, AlbumType: album.AlbumType,
			ReleaseDate: album.ReleaseDate, TotalTracks: album.TotalTracks,
			Artists: artists, Images: toModelImages(album.Images), SpotifyURL: album.ExternalURLs["spotify"],
		}
		for i := range album.Tracks.Items {
			track := album.Tracks.Items[i]
			track.Album = album.spotifyAlbum
			details.Tracks = append(details.Tracks, trackDetailsFromSpotify(&track))
		}
		albums = append(albums, details)
	}
	return albums, body, nil
}

// GetAlbumsBatchRaw fetches any number of album editions in batches of 20.
func GetAlbumsBatchRaw(c *client.Client, ids []string) ([]models.AlbumDetails, [][]byte, error) {
	var all []models.AlbumDetails
	var bodies [][]byte
	for i := 0; i < len(ids); i += 20 {
		end := min(i+20, len(ids))
		batch, body, err := GetAlbumsRaw(c, ids[i:end])
		if err != nil {
			return nil, nil, err
		}
		all = append(all, batch...)
		if len(body) > 0 {
			bodies = append(bodies, body)
		}
	}
	return all, bodies, nil
}

func trackDetailsFromSpotify(track *spotifyTrack) models.TrackDetails {
	if track == nil {
		return models.TrackDetails{}
	}
	artists := make([]models.PlayArtist, 0, len(track.Artists))
	for _, artist := range track.Artists {
		artists = append(artists, models.PlayArtist{
			ID:         artist.ID,
			Name:       artist.Name,
			SpotifyURL: artist.ExternalURLs["spotify"],
		})
	}
	albumArtists := make([]models.PlayArtist, 0, len(track.Album.Artists))
	for _, artist := range track.Album.Artists {
		albumArtists = append(albumArtists, models.PlayArtist{
			ID:         artist.ID,
			Name:       artist.Name,
			SpotifyURL: artist.ExternalURLs["spotify"],
		})
	}
	return models.TrackDetails{
		ID:               track.ID,
		Name:             track.Name,
		Artists:          artists,
		AlbumArtists:     models.NormalizeAlbumArtists(firstPlayArtist(artists), albumArtists),
		AlbumID:          track.Album.ID,
		AlbumName:        track.Album.Name,
		AlbumType:        track.Album.AlbumType,
		AlbumReleaseDate: track.Album.ReleaseDate,
		AlbumTotalTracks: track.Album.TotalTracks,
		AlbumImages:      toModelImages(track.Album.Images),
		DiscNumber:       track.DiscNumber,
		TrackNumber:      track.TrackNumber,
		DurationMS:       track.DurationMS,
		TrackSpotifyURL:  track.ExternalURLs["spotify"],
	}
}

func firstPlayArtist(artists []models.PlayArtist) models.PlayArtist {
	if len(artists) == 0 {
		return models.PlayArtist{}
	}
	return artists[0]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- Setlist.fm ---

// setlist.fm API response types (internal use only).
type setlistfmResponse struct {
	Setlist []setlistfmSetlist `json:"setlist"`
}

type setlistfmSetlist struct {
	EventDate string            `json:"eventDate"` // "DD-MM-YYYY"
	Artist    setlistfmArtist   `json:"artist"`
	Venue     setlistfmVenue    `json:"venue"`
	URL       string            `json:"url"`
	Sets      setlistfmSetsCont `json:"sets"`
}

type setlistfmArtist struct {
	Name string `json:"name"`
}

type setlistfmVenue struct {
	Name string        `json:"name"`
	City setlistfmCity `json:"city"`
}

type setlistfmCity struct {
	Name      string           `json:"name"`
	StateCode string           `json:"stateCode"`
	Country   setlistfmCountry `json:"country"`
}

type setlistfmCountry struct {
	Code string `json:"code"`
}

type setlistfmSetsCont struct {
	Set []setlistfmSet `json:"set"`
}

type setlistfmSet struct {
	Name string          `json:"name"` // "" for main set, "Encore" etc.
	Song []setlistfmSong `json:"song"`
}

type setlistfmSong struct {
	Name string `json:"name"`
}

// setlistGet performs a GET request to the setlist.fm REST API.
func setlistGet(path string, params url.Values) ([]byte, error) {
	apiKey := os.Getenv("SETLISTFM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("SETLISTFM_API_KEY not set")
	}

	reqURL := "https://api.setlist.fm/rest/1.0" + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("setlist.fm build request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("setlist.fm request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("setlist.fm read body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no setlist found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("setlist.fm returned %d", resp.StatusCode)
	}
	return body, nil
}

// GetSetlist fetches the most recent setlist for artistName on date (YYYY-MM-DD).
// Returns the first matching result from setlist.fm.
func GetSetlist(artistName, date string) (models.Setlist, error) {
	// Convert YYYY-MM-DD to DD-MM-YYYY for setlist.fm
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return models.Setlist{}, fmt.Errorf("invalid date %q: %w", date, err)
	}
	apiDate := t.Format("02-01-2006")

	params := url.Values{}
	params.Set("artistName", artistName)
	params.Set("date", apiDate)
	params.Set("p", "1")

	body, err := setlistGet("/search/setlists", params)
	if err != nil {
		return models.Setlist{}, err
	}

	var resp setlistfmResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return models.Setlist{}, fmt.Errorf("setlist.fm decode: %w", err)
	}

	if len(resp.Setlist) == 0 {
		return models.Setlist{}, fmt.Errorf("no setlist found for %q on %s", artistName, date)
	}

	raw := resp.Setlist[0]

	city := raw.Venue.City.Name
	if raw.Venue.City.StateCode != "" {
		city += ", " + raw.Venue.City.StateCode
	}

	var sets []models.SetlistSet
	for _, s := range raw.Sets.Set {
		var songs []string
		for _, song := range s.Song {
			songs = append(songs, song.Name)
		}
		sets = append(sets, models.SetlistSet{
			Name:  s.Name,
			Songs: songs,
		})
	}

	return models.Setlist{
		EventDate:  raw.EventDate,
		ArtistName: raw.Artist.Name,
		VenueName:  raw.Venue.Name,
		CityName:   city,
		URL:        raw.URL,
		Sets:       sets,
	}, nil
}

// GetTopArtists fetches the user's top 50 artists for the given time range.
// timeRange: "short_term" | "medium_term" | "long_term"
func GetTopArtists(c *client.Client, timeRange string) ([]models.TopArtist, error) {
	artists, _, err := GetTopArtistsRaw(c, timeRange)
	return artists, err
}

// GetTopArtistsRaw fetches the user's top artists and returns the unchanged
// Spotify response body alongside mapped artists.
func GetTopArtistsRaw(c *client.Client, timeRange string) ([]models.TopArtist, []byte, error) {
	params := url.Values{}
	params.Set("limit", "50")
	params.Set("time_range", timeRange)

	body, err := c.Get("/me/top/artists", params)
	if err != nil {
		return nil, nil, fmt.Errorf("top/artists: %w", err)
	}

	var resp topArtistsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("top/artists decode: %w", err)
	}

	artists := make([]models.TopArtist, 0, len(resp.Items))
	for _, item := range resp.Items {
		artists = append(artists, models.TopArtist{
			ID:         item.ID,
			Name:       item.Name,
			Genres:     item.Genres,
			SpotifyURL: item.ExternalURLs["spotify"],
			Images:     toModelImages(item.Images),
		})
	}
	return artists, body, nil
}
