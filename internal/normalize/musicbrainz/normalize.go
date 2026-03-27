package musicbrainznormalize

import (
	"github.com/benstraw/music-garden/internal/clients/musicbrainz"
	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/genres"
)

// ArtistSeed is the Spotify-facing seed used to enrich a canonical artist.
type ArtistSeed struct {
	SpotifyArtistID string
	Name            string
	SpotifyURL      string
}

// ReleaseSeed is the Spotify-facing seed used to enrich a canonical release.
type ReleaseSeed struct {
	SpotifyAlbumID       string
	Name                 string
	PrimaryArtistName    string
	PrimaryArtistID      string
	PrimaryArtistSpotify string
}

// NormalizeArtist converts one MusicBrainz artist into normalized and canonical records.
func NormalizeArtist(store *genres.Store, seed ArtistSeed, artist musicbrainz.Artist) (datalayer.NormalizedArtistRecord, genres.ArtistRecord) {
	sourceGenres := musicbrainz.GenreNames(artist.Genres, artist.Tags)
	record := genres.UpsertArtistMetadata(store, seed.SpotifyArtistID, firstNonEmpty(seed.Name, artist.Name), seed.SpotifyURL, artist.ID, sourceGenres, nil)
	normalized := datalayer.NormalizedArtistRecord{
		Source:              "musicbrainz",
		SourceArtistID:      artist.ID,
		Name:                firstNonEmpty(artist.Name, seed.Name),
		SpotifyURL:          seed.SpotifyURL,
		MusicBrainzArtistID: artist.ID,
		SourceGenres:        sourceGenres,
		CanonicalGenreSlugs: append([]string(nil), record.Genres...),
	}
	return normalized, record
}

// NormalizeRelease converts one MusicBrainz release-group into normalized and canonical records.
func NormalizeRelease(store *genres.Store, seed ReleaseSeed, group musicbrainz.ReleaseGroup) (datalayer.NormalizedReleaseRecord, genres.ReleaseRecord, genres.ArtistRecord, []datalayer.NormalizedGenreRecord) {
	artistName := firstNonEmpty(musicbrainz.PrimaryArtistName(group.ArtistCredit), seed.PrimaryArtistName)
	artist := genres.UpsertArtistMetadata(store, seed.PrimaryArtistID, artistName, seed.PrimaryArtistSpotify, "", musicbrainz.GenreNames(group.Genres, group.Tags), nil)
	releaseID := ""
	if len(group.Releases) > 0 {
		releaseID = group.Releases[0].ID
	}
	record := genres.UpsertReleaseMetadata(store, artist, seed.SpotifyAlbumID, firstNonEmpty(seed.Name, group.Title), group.ID, releaseID)
	normalized := datalayer.NormalizedReleaseRecord{
		Source:                     "musicbrainz",
		SourceReleaseID:            group.ID,
		Name:                       firstNonEmpty(group.Title, seed.Name),
		PrimaryArtistName:          artist.Name,
		PrimaryArtistCanonicalSlug: artist.Slug,
		MusicBrainzReleaseID:       releaseID,
		MusicBrainzReleaseGroupID:  group.ID,
	}
	sourceGenres := musicbrainz.GenreNames(group.Genres, group.Tags)
	genreRecords := make([]datalayer.NormalizedGenreRecord, 0, len(sourceGenres))
	for _, sourceGenre := range sourceGenres {
		canonical, ok := genres.CanonicalGenre(store, sourceGenre)
		record := datalayer.NormalizedGenreRecord{
			Source:      "musicbrainz",
			SourceGenre: sourceGenre,
		}
		if ok {
			record.CanonicalGenreSlug = canonical
		}
		genreRecords = append(genreRecords, record)
	}
	return normalized, record, artist, genreRecords
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
