package musicbrainznormalize

import (
	"testing"

	"github.com/benstraw/music-garden/internal/clients/musicbrainz"
	"github.com/benstraw/music-garden/internal/genres"
)

func TestNormalizeArtist_setsMusicBrainzIDAndGenres(t *testing.T) {
	store := genres.NewStore()
	genres.ApplyTaxonomy(store, &genres.Taxonomy{
		Version: 1,
		Genres:  []genres.GenreDefinition{{Slug: "ambient", DisplayName: "Ambient", Aliases: []string{"ambient"}}},
	})

	normalized, record := NormalizeArtist(store, ArtistSeed{
		SpotifyArtistID: "spotify-artist-1",
		Name:            "Biosphere",
		SpotifyURL:      "https://open.spotify.com/artist/spotify-artist-1",
	}, musicbrainz.Artist{
		ID:   "mbid-artist-1",
		Name: "Biosphere",
		Genres: []musicbrainz.TagCount{
			{Name: "ambient"},
		},
	})

	if normalized.Source != "musicbrainz" {
		t.Fatalf("Source = %q", normalized.Source)
	}
	if record.MusicBrainzArtistID != "mbid-artist-1" {
		t.Fatalf("MusicBrainzArtistID = %q", record.MusicBrainzArtistID)
	}
	if len(record.Genres) != 1 || record.Genres[0] != "ambient" {
		t.Fatalf("Genres = %v", record.Genres)
	}
}

func TestNormalizeRelease_setsReleaseGroupAndGenres(t *testing.T) {
	store := genres.NewStore()
	genres.ApplyTaxonomy(store, &genres.Taxonomy{
		Version: 1,
		Genres:  []genres.GenreDefinition{{Slug: "electronic", DisplayName: "Electronic", Aliases: []string{"electronic"}}},
	})

	normalized, releaseRecord, artistRecord, genreRecords := NormalizeRelease(store, ReleaseSeed{
		SpotifyAlbumID:       "spotify-album-1",
		Name:                 "Moon Safari",
		PrimaryArtistName:    "Air",
		PrimaryArtistID:      "spotify-artist-1",
		PrimaryArtistSpotify: "https://open.spotify.com/artist/spotify-artist-1",
	}, musicbrainz.ReleaseGroup{
		ID:    "mbid-rg-1",
		Title: "Moon Safari",
		ArtistCredit: []musicbrainz.ArtistCredit{
			{Name: "Air", Artist: musicbrainz.CreditArtist{ID: "mbid-artist-1", Name: "Air"}},
		},
		Genres: []musicbrainz.TagCount{{Name: "electronic"}},
		Releases: []musicbrainz.ReleaseRef{
			{ID: "mbid-release-1", Title: "Moon Safari"},
		},
	})

	if normalized.MusicBrainzReleaseGroupID != "mbid-rg-1" {
		t.Fatalf("MusicBrainzReleaseGroupID = %q", normalized.MusicBrainzReleaseGroupID)
	}
	if releaseRecord.MusicBrainzReleaseGroupID != "mbid-rg-1" {
		t.Fatalf("releaseRecord.MusicBrainzReleaseGroupID = %q", releaseRecord.MusicBrainzReleaseGroupID)
	}
	if artistRecord.Name != "Air" {
		t.Fatalf("artistRecord.Name = %q", artistRecord.Name)
	}
	if len(genreRecords) != 1 || genreRecords[0].CanonicalGenreSlug != "electronic" {
		t.Fatalf("genreRecords = %+v", genreRecords)
	}
}
