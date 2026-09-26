package ai

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestJSONSchemaRejectsLengthAndDuplicateSymbols(t *testing.T) {
	document := GroundingDocument{URL: "https://example.invalid/item", Source: "fixture", Title: "Update", Text: "Evidence", PublishedAt: time.Now().UTC()}
	analysis := validAnalysis(document)
	analysis.Summary = strings.Repeat("x", 1201)
	raw, _ := json.Marshal(analysis)
	if err := ValidateJSONSchema(raw); err == nil {
		t.Fatal("expected maximum length rejection")
	}
	analysis.Summary = "Update"
	analysis.RelevantSymbols = []string{"RELIANCE", "RELIANCE"}
	raw, _ = json.Marshal(analysis)
	if err := ValidateJSONSchema(raw); err == nil {
		t.Fatal("expected duplicate symbol rejection")
	}
}

func TestEmbeddedSchemaMatchesPublishedContract(t *testing.T) {
	published, err := os.ReadFile("../../docs/ai-output.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var publicSchema map[string]any
	if err := json.Unmarshal(published, &publicSchema); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(publicSchema, analysisSchema) {
		t.Fatal("embedded validator schema drifted from docs/ai-output.schema.json")
	}
}
