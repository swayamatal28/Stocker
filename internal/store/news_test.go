package store

import "testing"

func TestJaccardDetectsNearDuplicate(t *testing.T) {
	left := "Infosys announces a new cloud services agreement with a banking customer"
	right := "Infosys announces new cloud services agreement with banking customer"
	if score := jaccard(left, right); score < 0.8 {
		t.Fatalf("expected a near duplicate score, got %.2f", score)
	}
}

func TestJaccardRejectsUnrelatedStories(t *testing.T) {
	if score := jaccard("bank raises deposit rates", "manufacturer opens solar factory"); score != 0 {
		t.Fatalf("expected unrelated stories to score zero, got %.2f", score)
	}
}
