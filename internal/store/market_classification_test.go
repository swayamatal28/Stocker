package store

import "testing"

func TestUsefulClassificationRejectsProviderPlaceholders(t *testing.T) {
	for _, value := range []string{"", " ", "Unclassified", "UNKNOWN", "N/A", "not available", "Other"} {
		if usefulClassification(value) {
			t.Fatalf("expected %q to be treated as missing classification", value)
		}
	}
	for _, value := range []string{"Information Technology", "Basic Materials", "Auto Manufacturers"} {
		if !usefulClassification(value) {
			t.Fatalf("expected %q to be retained", value)
		}
	}
}
