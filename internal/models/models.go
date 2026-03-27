package models

// Play represents a single music track play event.
// Spotify is the current upstream source, but canonical artist/release IDs are
// garden-owned so downstream code does not depend on one provider's taxonomy.
type Play struct {
	PlayedAt                  string `json:"played_at"`
	Source                    string `json:"source,omitempty"`
	TrackID                   string `json:"track_id"`
	TrackName                 string `json:"track_name"`
	ArtistSlug                string `json:"artist_slug,omitempty"`
	ArtistID                  string `json:"artist_id"`
	ArtistName                string `json:"artist_name"`
	ArtistSpotifyURL          string `json:"artist_spotify_url"`
	ArtistMusicBrainzID       string `json:"artist_musicbrainz_id,omitempty"`
	ReleaseSlug               string `json:"release_slug,omitempty"`
	AlbumID                   string `json:"album_id,omitempty"`
	AlbumName                 string `json:"album_name"`
	ReleaseMusicBrainzID      string `json:"release_musicbrainz_id,omitempty"`
	ReleaseGroupMusicBrainzID string `json:"release_group_musicbrainz_id,omitempty"`
	DurationMS                int    `json:"duration_ms"`
	TrackSpotifyURL           string `json:"track_spotify_url"`
}

// TopTrack represents a track from the user's top tracks.
type TopTrack struct {
	ID         string
	Name       string
	ArtistName string
}

// ArtistImage represents one size of a Spotify artist profile image.
type ArtistImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// TopArtist represents an artist from the user's top artists.
type TopArtist struct {
	ID                  string
	Name                string
	ArtistSlug          string
	MusicBrainzArtistID string
	Genres              []string
	SourceGenres        []string
	SpotifyURL          string
	Images              []ArtistImage
}

// Setlist represents a setlist.fm setlist result.
type Setlist struct {
	EventDate  string // "DD-MM-YYYY" from API
	ArtistName string
	VenueName  string
	CityName   string
	URL        string // setlist.fm URL
	Sets       []SetlistSet
}

// SetlistSet represents one set (main set or encore) within a setlist.
type SetlistSet struct {
	Name  string // "Encore", "" for main set
	Songs []string
}
