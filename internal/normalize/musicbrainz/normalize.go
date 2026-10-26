package musicbrainznormalize

import (
	"github.com/benstraw/music-garden/internal/clients/musicbrainz"
	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/genres"
	"strings"
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
func NormalizeArtist(source any, seed ArtistSeed, artist musicbrainz.Artist) (datalayer.NormalizedArtistRecord, genres.ArtistRecord) {
	catalog := ensureCatalog(source)
	sourceGenres := musicbrainz.GenreNames(artist.Genres, artist.Tags)
	record := genres.UpsertArtistMetadata(catalog, seed.SpotifyArtistID, firstNonEmpty(seed.Name, artist.Name), seed.SpotifyURL, artist.ID, sourceGenres, nil)
	record.SortName = artist.SortName
	record.ArtistType = artist.Type
	record.Country = artist.Country
	record.LifeSpan = &genres.LifeSpan{Begin: artist.LifeSpan.Begin, End: artist.LifeSpan.End, Ended: artist.LifeSpan.Ended}
	if record.LifeSpan.Begin == "" && record.LifeSpan.End == "" && !record.LifeSpan.Ended {
		record.LifeSpan = nil
	}
	for _, alias := range artist.Aliases {
		if alias.Name != "" {
			record.Aliases = append(record.Aliases, alias.Name)
		}
	}
	for _, relation := range artist.Relations {
		if relation.Artist != nil {
			record.Relationships = append(record.Relationships, genres.ArtistRelationship{
				Type: relation.Type, Direction: relation.Direction,
				TargetSlug: genres.CanonicalArtistSlugForRecord(relation.Artist.Name),
				TargetName: relation.Artist.Name, TargetMBID: relation.Artist.ID,
				Begin: relation.Begin, End: relation.End,
				SourceURL: "https://musicbrainz.org/artist/" + artist.ID,
			})
		}
		if relation.URL != nil && relation.Type == "wikidata" {
			parts := strings.Split(strings.TrimRight(relation.URL.Resource, "/"), "/")
			record.WikidataID = parts[len(parts)-1]
		}
	}
	record = genres.ReplaceArtistRecord(catalog, record)
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
func NormalizeRelease(source any, seed ReleaseSeed, group musicbrainz.ReleaseGroup) (datalayer.NormalizedReleaseRecord, genres.ReleaseRecord, genres.ArtistRecord, []datalayer.NormalizedGenreRecord) {
	catalog := ensureCatalog(source)
	artistName := firstNonEmpty(musicbrainz.PrimaryArtistName(group.ArtistCredit), seed.PrimaryArtistName)
	artist := genres.UpsertArtistMetadata(catalog, seed.PrimaryArtistID, artistName, seed.PrimaryArtistSpotify, "", musicbrainz.GenreNames(group.Genres, group.Tags), nil)
	releaseID := ""
	if len(group.Releases) > 0 {
		releaseID = group.Releases[0].ID
	}
	record := genres.UpsertReleaseMetadata(catalog, artist, seed.SpotifyAlbumID, firstNonEmpty(seed.Name, group.Title), group.ID, releaseID)
	record.PrimaryType = group.PrimaryType
	record.SecondaryTypes = append([]string(nil), group.SecondaryTypes...)
	record.FirstReleaseDate = group.FirstReleaseDate
	record.ArtistCredits = make([]genres.PlayArtistRef, 0, len(group.ArtistCredit))
	for _, credit := range group.ArtistCredit {
		record.ArtistCredits = append(record.ArtistCredits, genres.PlayArtistRef{
			Slug: genres.CanonicalArtistSlugForRecord(credit.Artist.Name), Name: firstNonEmpty(credit.Name, credit.Artist.Name), MBID: credit.Artist.ID,
		})
	}
	record = genres.ReplaceReleaseRecord(catalog, record)
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
		canonical, ok := genres.CanonicalGenre(catalog.Genres, sourceGenre)
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

func ensureCatalog(source any) *genres.Catalog {
	switch v := source.(type) {
	case *genres.Catalog:
		return v
	case *genres.Store:
		catalog := genres.NewCatalog()
		catalog.Genres = v
		catalog.Artists.Artists = v.Artists
		catalog.Artists.ArtistSlugAliases = v.ArtistSlugAliases
		catalog.Artists.ArtistSourceIndex = v.ArtistSourceIndex
		catalog.Releases.Releases = v.Releases
		catalog.Releases.ReleaseSourceIndex = v.ReleaseSourceIndex
		return catalog
	default:
		panic("unsupported catalog source")
	}
}
