package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPProvider struct {
	endpoint, apiKey, model   string
	client                    *http.Client
	maxOutputBytes, costCents int
}

func NewHTTPProvider(endpoint, apiKey, model string, timeout time.Duration, maxOutputBytes, costCents int) (*HTTPProvider, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("AI endpoint must be an absolute HTTP(S) URL")
	}
	if model == "" {
		return nil, errors.New("AI model is required")
	}
	if maxOutputBytes < 1024 {
		maxOutputBytes = 64 << 10
	}
	return &HTTPProvider{endpoint: endpoint, apiKey: apiKey, model: model, client: &http.Client{Timeout: timeout}, maxOutputBytes: maxOutputBytes, costCents: max(costCents, 0)}, nil
}
func (provider *HTTPProvider) Name() string                           { return "http-json" }
func (provider *HTTPProvider) Model() string                          { return provider.model }
func (provider *HTTPProvider) EstimatedCostCents(GroundedRequest) int { return provider.costCents }
func (provider *HTTPProvider) Analyze(ctx context.Context, request GroundedRequest) (json.RawMessage, error) {
	payload, err := json.Marshal(map[string]any{"model": provider.model, "request": request})
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if provider.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
	}
	response, err := provider.client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(provider.maxOutputBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > provider.maxOutputBytes {
		return nil, errors.New("AI provider response exceeded configured limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("AI provider returned status %d", response.StatusCode)
	}
	var envelope struct {
		Output json.RawMessage `json:"output"`
	}
	if json.Unmarshal(body, &envelope) == nil && len(envelope.Output) > 0 {
		return envelope.Output, nil
	}
	return body, nil
}

func (provider *HTTPProvider) Translate(ctx context.Context, text, sourceLanguage, targetLanguage string) (string, error) {
	payload, err := json.Marshal(map[string]any{"model": provider.model, "task": "translate", "sourceLanguage": sourceLanguage, "targetLanguage": targetLanguage, "text": text})
	if err != nil {
		return "", err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if provider.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
	}
	response, err := provider.client.Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("translation provider returned status %d", response.StatusCode)
	}
	var result struct {
		Translation string `json:"translation"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, int64(provider.maxOutputBytes))).Decode(&result); err != nil {
		return "", err
	}
	if result.Translation == "" {
		return "", errors.New("translation provider returned empty text")
	}
	return result.Translation, nil
}
