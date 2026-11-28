package mediawikinormalize

import (
	"testing"
	"time"

	"github.com/benstraw/music-garden/internal/clients/mediawiki"
)

func TestNormalizeMatchedGenre(t *testing.T) {
	normalized, record := NormalizeMatchedGenre(GenreSeed{
		CanonicalSlug: "ambient",
		SearchTerm:    "ambient music",
	}, mediawiki.Summary{
		Title:   "Ambient music",
		Extract: "Ambient music is a genre of music.",
		ContentURLs: mediawiki.ContentURLs{
			Desktop: mediawiki.PageURLs{Page: "https://en.wikipedia.org/wiki/Ambient_music"},
		},
		Thumbnail: &mediawiki.Image{Source: "https://upload.wikimedia.org/thumb.jpg", Width: 320, Height: 180},
	}, []string{"Ambient music"}, []mediawiki.ImageInfo{{
		FileTitle:    "File:Ambient_music.jpg",
		FilePageURL:  "https://commons.wikimedia.org/wiki/File:Ambient_music.jpg",
		ImageURL:     "https://upload.wikimedia.org/original.jpg",
		ThumbnailURL: "https://upload.wikimedia.org/thumb.jpg",
		Width:        1200,
		Height:       800,
		License:      "CC BY-SA 4.0",
	}}, time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC))

	if normalized.Status != "matched" {
		t.Fatalf("Status = %q", normalized.Status)
	}
	if record.WikipediaTitle != "Ambient music" || record.Image == nil {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestNormalizeUnresolvedGenre(t *testing.T) {
	normalized, record := NormalizeUnresolvedGenre(GenreSeed{CanonicalSlug: "rock"}, "ambiguous", []string{"Rock music", "Rock"}, time.Now().UTC())
	if normalized.Status != "ambiguous" || record.Status != "ambiguous" {
		t.Fatalf("unexpected statuses: %q %q", normalized.Status, record.Status)
	}
	if len(record.Candidates) != 2 {
		t.Fatalf("Candidates = %+v", record.Candidates)
	}
}

func TestNormalizeMatchedArtist(t *testing.T) {
	normalized, record := NormalizeMatchedArtist(ArtistSeed{
		CanonicalSlug: "radiohead",
		Name:          "Radiohead",
		SearchTerm:    "Radiohead",
	}, mediawiki.Summary{
		Title:   "Radiohead",
		Extract: "Radiohead are an English rock band.",
		ContentURLs: mediawiki.ContentURLs{
			Desktop: mediawiki.PageURLs{Page: "https://en.wikipedia.org/wiki/Radiohead"},
		},
	}, []string{"Radiohead"}, []mediawiki.ImageInfo{{
		FileTitle: "File:Radiohead.jpg",
		ImageURL:  "https://upload.wikimedia.org/original.jpg",
	}}, time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC))

	if normalized.Status != "matched" {
		t.Fatalf("Status = %q", normalized.Status)
	}
	if record.WikipediaTitle != "Radiohead" || record.Image == nil {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestNormalizeUnresolvedArtist(t *testing.T) {
	normalized, record := NormalizeUnresolvedArtist(ArtistSeed{CanonicalSlug: "xtc", Name: "XTC"}, "ambiguous", []string{"XTC", "XTC (band)"}, time.Now().UTC())
	if normalized.Status != "ambiguous" || record.Status != "ambiguous" {
		t.Fatalf("unexpected statuses: %q %q", normalized.Status, record.Status)
	}
}
