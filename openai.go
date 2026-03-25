package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

// SupportedOpenAIModels is the allow-list of model names fmc accepts.
var SupportedOpenAIModels = []string{
	"gpt-4o",
	"gpt-4o-mini",
	"gpt-4-turbo",
	"gpt-3.5-turbo",
}

// GeneratedFields holds the structured output returned by the LLM for one file.
type GeneratedFields struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Keywords    []string `json:"keywords"`
}

// openAIRequest is the body sent to the Chat Completions endpoint.
type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	ResponseFormat responseFormat  `json:"response_format"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *jsonSchema `json:"json_schema,omitempty"`
}

type jsonSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

// openAIResponse covers the fields we care about from the API.
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// fieldSchema builds the JSON schema for the requested field names.
// Fields not in the known set are silently ignored.
func fieldSchema(fields []string) map[string]any {
	props := map[string]any{}
	required := []string{}

	for _, f := range fields {
		switch f {
		case "title", "description":
			props[f] = map[string]any{"type": "string"}
			required = append(required, f)
		case "tags", "keywords":
			props[f] = map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			}
			required = append(required, f)
		}
	}

	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// systemPrompt returns a system message tailored to the fields being requested.
func systemPrompt(fields []string) string {
	var parts []string
	for _, f := range fields {
		switch f {
		case "title":
			parts = append(parts, `"title": a concise, descriptive page title (plain text, no markdown)`)
		case "description":
			parts = append(parts, `"description": 1-2 sentence SEO-friendly summary of the page content (plain text)`)
		case "tags":
			parts = append(parts, `"tags": broad navigation categories as a JSON array of lowercase strings (e.g. ["go","tutorial"])`)
		case "keywords":
			parts = append(parts, `"keywords": specific SEO keyword phrases as a JSON array of lowercase strings`)
		}
	}
	return "You are a technical documentation assistant. " +
		"Analyze the provided markdown document and return ONLY a JSON object with these fields:\n" +
		strings.Join(parts, "\n") + "\n\n" +
		"Return nothing outside the JSON object."
}

// GenerateFields calls the OpenAI Chat Completions API with structured outputs
// and returns the parsed response. Only the fields listed in wantFields are
// requested; the rest are omitted from the schema.
func GenerateFields(apiKey, model string, wantFields []string, markdownContent string) (GeneratedFields, error) {
	if err := validateModel(model); err != nil {
		return GeneratedFields{}, err
	}

	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt(wantFields)},
			{Role: "user", Content: markdownContent},
		},
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchema{
				Name:   "front_matter_fields",
				Strict: true,
				Schema: fieldSchema(wantFields),
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, openAIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("calling OpenAI: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeneratedFields{}, fmt.Errorf("reading response body: %w", err)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(rawBody, &apiResp); err != nil {
		return GeneratedFields{}, fmt.Errorf("parsing response (status %d): %w\nbody: %s", resp.StatusCode, err, rawBody)
	}

	// OpenAI returns errors either as non-2xx status or as a 200 with an error object.
	if apiResp.Error != nil {
		return GeneratedFields{}, fmt.Errorf("OpenAI error [%s/%s]: %s", apiResp.Error.Type, apiResp.Error.Code, apiResp.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GeneratedFields{}, fmt.Errorf("OpenAI returned HTTP %d: %s", resp.StatusCode, rawBody)
	}

	if len(apiResp.Choices) == 0 {
		return GeneratedFields{}, fmt.Errorf("OpenAI returned no choices")
	}

	choice := apiResp.Choices[0]
	switch choice.FinishReason {
	case "content_filter":
		return GeneratedFields{}, fmt.Errorf("OpenAI content filter triggered — skipping file")
	case "length":
		// Structured outputs with strict mode should not truncate, but handle it defensively.
		return GeneratedFields{}, fmt.Errorf("OpenAI response truncated (finish_reason: length)")
	}

	var result GeneratedFields
	if err := json.Unmarshal([]byte(choice.Message.Content), &result); err != nil {
		return GeneratedFields{}, fmt.Errorf("parsing structured output: %w\ncontent: %s", err, choice.Message.Content)
	}
	return result, nil
}

func validateModel(model string) error {
	for _, m := range SupportedOpenAIModels {
		if m == model {
			return nil
		}
	}
	return fmt.Errorf("unsupported OpenAI model %q — supported: %s", model, strings.Join(SupportedOpenAIModels, ", "))
}

// runLLMTest verifies the config, model, and API key by sending a minimal
// request and reporting exactly what passed or failed.
func runLLMTest() {
	cfg, err := LoadFMCConfig()
	if err != nil {
		fmt.Printf("FAIL  config: %v\n", err)
		return
	}
	fmt.Println("OK    config loaded from ~/.fmc/config.json")

	if cfg.OpenAI.APIKey == "" {
		fmt.Println("FAIL  openai.api_key is not set")
		return
	}
	fmt.Println("OK    api_key present")

	model := cfg.OpenAI.Model
	if model == "" {
		model = "gpt-4o"
		fmt.Printf("note  openai.model not set, using default %q\n", model)
	}

	if err := validateModel(model); err != nil {
		fmt.Printf("FAIL  %v\n", err)
		return
	}
	fmt.Printf("OK    model %q is supported\n", model)

	fmt.Printf("...   sending test request to OpenAI...\n")

	// Minimal request — ask for a single short string to keep token cost near zero.
	_, err = GenerateFields(cfg.OpenAI.APIKey, model, []string{"title"}, "ping")
	if err != nil {
		fmt.Printf("FAIL  API call failed: %v\n", err)
		return
	}
	fmt.Println("OK    API call succeeded — connection is working")
}
