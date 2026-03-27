package genres

import (
	"path/filepath"
	"testing"
)

func TestResolveGenreLabel(t *testing.T) {
	taxonomy := &Taxonomy{
		Version: 1,
		Genres: []GenreDefinition{
			{Slug: "hip-hop", DisplayName: "Hip-Hop", Aliases: []string{"hip hop", "hiphop"}},
		},
	}

	got, ok, ambiguous := ResolveGenreLabel(taxonomy, "Hip Hop")
	if !ok || len(ambiguous) > 0 || got != "hip-hop" {
		t.Fatalf("ResolveGenreLabel = (%q,%v,%v)", got, ok, ambiguous)
	}
}

func TestBuildGenreReport_detectsCollisionsAndUnknowns(t *testing.T) {
	taxonomy := &Taxonomy{
		Version: 1,
		Genres: []GenreDefinition{
			{Slug: "hip-hop", DisplayName: "Hip-Hop", Aliases: []string{"hip hop"}},
			{Slug: "rap", DisplayName: "Rap", Aliases: []string{"hip hop"}},
		},
	}

	report := BuildGenreReport(taxonomy, []string{"hip hop", "space jazz"})
	if len(report.Collisions) != 1 {
		t.Fatalf("Collisions = %+v", report.Collisions)
	}
	if len(report.UnknownLabels) != 1 || report.UnknownLabels[0] != "space jazz" {
		t.Fatalf("UnknownLabels = %+v", report.UnknownLabels)
	}
}

func TestLoadTaxonomy(t *testing.T) {
	path := filepath.Join("..", "..", "data", "genre-taxonomy.json")
	taxonomy, err := LoadTaxonomy(path)
	if err != nil {
		t.Fatalf("LoadTaxonomy: %v", err)
	}
	if len(taxonomy.Genres) == 0 {
		t.Fatal("expected curated genres")
	}
}
