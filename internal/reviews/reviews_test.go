package reviews

import "testing"

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("artist", "the-beatles", Decision{Status: "approved", ForcePublish: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveKind(dir, "artist"); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	decision, ok := loaded.Decision("artists", "the-beatles")
	if !ok || !decision.ForcePublish || decision.Status != "approved" {
		t.Fatalf("decision = %+v, ok=%v", decision, ok)
	}
}

func TestStoreRejectsUnknownKind(t *testing.T) {
	store, _ := Load(t.TempDir())
	if err := store.Set("track", "song", Decision{Status: "approved"}); err == nil {
		t.Fatal("expected unsupported kind error")
	}
}
