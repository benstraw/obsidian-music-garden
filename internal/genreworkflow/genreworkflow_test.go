package genreworkflow

import (
	"testing"

	"github.com/benstraw/music-garden/internal/datalayer"
	"github.com/benstraw/music-garden/internal/genres"
)

func TestQueueCandidates_ordersByPlayCountThenEnrichmentThenSlug(t *testing.T) {
	views := []GenreView{
		{
			Slug:          "beta",
			Mapped:        true,
			WorkflowState: genres.WorkflowStateDraft,
			ListeningStats: datalayer.GenreListeningStats{
				PlayCount: 5,
			},
			Summary:    "present",
			HasImage:   true,
			ParentSlug: "root",
		},
		{
			Slug:          "alpha",
			Mapped:        true,
			WorkflowState: genres.WorkflowStateDraft,
			ListeningStats: datalayer.GenreListeningStats{
				PlayCount: 5,
			},
			Summary: "present",
		},
		{
			Slug:          "gamma",
			Mapped:        true,
			WorkflowState: genres.WorkflowStateDraft,
			ListeningStats: datalayer.GenreListeningStats{
				PlayCount: 7,
			},
		},
		{
			Slug:          "publishable",
			Mapped:        true,
			WorkflowState: genres.WorkflowStatePublishable,
			ListeningStats: datalayer.GenreListeningStats{
				PlayCount: 99,
			},
		},
		{
			Slug:          "pending",
			Pending:       true,
			Mapped:        false,
			WorkflowState: genres.WorkflowStateDraft,
			ListeningStats: datalayer.GenreListeningStats{
				PlayCount: 100,
			},
		},
	}

	got := QueueCandidates(views, 0)
	if len(got) != 3 {
		t.Fatalf("len(QueueCandidates) = %d, want 3", len(got))
	}
	if got[0].Slug != "gamma" || got[1].Slug != "beta" || got[2].Slug != "alpha" {
		t.Fatalf("queue order = %v", []string{got[0].Slug, got[1].Slug, got[2].Slug})
	}
}

func TestEligibleForRule_readyBasic(t *testing.T) {
	view := GenreView{
		Slug:          "indie-rock",
		Mapped:        true,
		WorkflowState: genres.WorkflowStateDraft,
		Summary:       "present",
		ListeningStats: datalayer.GenreListeningStats{
			PlayCount: 5,
		},
	}
	ok, err := EligibleForRule(view, RuleReadyBasic, 5)
	if err != nil {
		t.Fatalf("EligibleForRule: %v", err)
	}
	if !ok {
		t.Fatal("EligibleForRule = false, want true")
	}
}

func TestEligibleForRule_readyLooseRejectsPending(t *testing.T) {
	view := GenreView{
		Slug:          "unknown-genre",
		Pending:       true,
		Mapped:        false,
		WorkflowState: genres.WorkflowStateDraft,
		ListeningStats: datalayer.GenreListeningStats{
			PlayCount: 100,
		},
	}
	ok, err := EligibleForRule(view, RuleReadyLoose, 10)
	if err != nil {
		t.Fatalf("EligibleForRule: %v", err)
	}
	if ok {
		t.Fatal("EligibleForRule = true, want false")
	}
}

func TestEligibleForRule_anyPlay(t *testing.T) {
	view := GenreView{
		Slug:          "acid-jazz",
		Mapped:        true,
		WorkflowState: genres.WorkflowStateDraft,
		ListeningStats: datalayer.GenreListeningStats{
			PlayCount: 1,
		},
	}
	ok, err := EligibleForRule(view, RuleAnyPlay, 1)
	if err != nil {
		t.Fatalf("EligibleForRule: %v", err)
	}
	if !ok {
		t.Fatal("EligibleForRule = false, want true")
	}
}

func TestEligibleForRule_anyPlayRich(t *testing.T) {
	view := GenreView{
		Slug:          "acid-jazz",
		Mapped:        true,
		WorkflowState: genres.WorkflowStateDraft,
		Summary:       "present",
		HasImage:      true,
		ListeningStats: datalayer.GenreListeningStats{
			PlayCount: 1,
		},
	}
	ok, err := EligibleForRule(view, RuleAnyPlayRich, 1)
	if err != nil {
		t.Fatalf("EligibleForRule: %v", err)
	}
	if !ok {
		t.Fatal("EligibleForRule = false, want true")
	}
}

func TestEligibleForRule_noPlayRich(t *testing.T) {
	view := GenreView{
		Slug:          "acid-jazz",
		Mapped:        true,
		WorkflowState: genres.WorkflowStateDraft,
		Summary:       "present",
		HasImage:      true,
		ListeningStats: datalayer.GenreListeningStats{
			PlayCount: 0,
		},
	}
	ok, err := EligibleForRule(view, RuleNoPlayRich, 0)
	if err != nil {
		t.Fatalf("EligibleForRule: %v", err)
	}
	if !ok {
		t.Fatal("EligibleForRule = false, want true")
	}
}

func TestEligibleForRule_resolvedGenreWithPlaysIsEligible(t *testing.T) {
	view := GenreView{
		Slug:          "classical",
		Mapped:        true,
		Pending:       false,
		WorkflowState: genres.WorkflowStateDraft,
		Summary:       "Classical music is art music.",
		ListeningStats: datalayer.GenreListeningStats{
			PlayCount: 274,
		},
	}
	ok, err := EligibleForRule(view, RuleReadyBasic, 5)
	if err != nil {
		t.Fatalf("EligibleForRule: %v", err)
	}
	if !ok {
		t.Fatal("EligibleForRule = false, want true")
	}
}
