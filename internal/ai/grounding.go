package ai

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/stocker-app/stocker/internal/domain"
)

var numberPattern = regexp.MustCompile(`[-+]?\d[\d,]*(?:\.\d+)?%?`)

func VerifyGrounding(analysis domain.AIAnalysis, request GroundedRequest) error {
	allowedSymbols := map[string]struct{}{}
	for _, symbol := range request.AllowedSymbols {
		allowedSymbols[strings.ToUpper(symbol)] = struct{}{}
	}
	for _, symbol := range analysis.RelevantSymbols {
		if _, ok := allowedSymbols[strings.ToUpper(symbol)]; !ok {
			return fmt.Errorf("unlinked symbol %s", symbol)
		}
	}
	documentsByURL := map[string]GroundingDocument{}
	corpus := ""
	for _, document := range request.Documents {
		documentsByURL[document.URL] = document
		corpus += " " + document.Title + " " + document.Text
	}
	for _, evidence := range analysis.Evidence {
		document, ok := documentsByURL[evidence.URL]
		if !ok {
			return fmt.Errorf("evidence URL is not a grounding document: %s", evidence.URL)
		}
		if evidence.Source != document.Source {
			return errors.New("evidence source does not match grounding document")
		}
		if !evidence.PublishedAt.Equal(document.PublishedAt) {
			return errors.New("evidence publication time does not match grounding document")
		}
		if !containsFold(document.Title+" "+document.Text, evidence.Excerpt) {
			return errors.New("evidence excerpt is not present in the grounding document")
		}
	}
	claims := []string{analysis.Summary, analysis.RetailExplanation, analysis.SectorImpact}
	claims = append(claims, analysis.SupportingFacts...)
	claims = append(claims, analysis.SecondOrderEffects...)
	for _, claim := range claims {
		for _, token := range numberPattern.FindAllString(claim, -1) {
			if numberPresent(token, corpus) || numberAllowed(token, request.AllowedNumericFacts) {
				continue
			}
			return fmt.Errorf("numeric claim %q is not grounded", token)
		}
	}
	return nil
}

func containsFold(haystack, needle string) bool {
	return needle != "" && strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
func numberPresent(token, text string) bool {
	compact := strings.ReplaceAll(token, ",", "")
	for _, candidate := range numberPattern.FindAllString(text, -1) {
		if strings.ReplaceAll(candidate, ",", "") == compact {
			return true
		}
	}
	return false
}
func numberAllowed(token string, facts map[string]float64) bool {
	clean := strings.TrimSuffix(strings.ReplaceAll(token, ",", ""), "%")
	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return false
	}
	for _, allowed := range facts {
		if fmt.Sprintf("%.6g", value) == fmt.Sprintf("%.6g", allowed) {
			return true
		}
	}
	return false
}
