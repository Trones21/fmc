package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FMCConfig is the user-level config loaded from ~/.fmc/config.json.
// It is separate from the per-run Config (policy/valueInsertion) already on
// FrontMatterChecker — that one is vestigial and project-scoped.
type FMCConfig struct {
	OpenAI OpenAIConfig `json:"openai"`
	LLM    LLMConfig    `json:"llm"`
}

type OpenAIConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"` // e.g. "gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"
}

// LLMConfig holds user preferences that apply to all LLM source generation.
type LLMConfig struct {
	// ContentDateField is the dotted front matter key whose value is compared
	// against date_last_generated when -llmRegenerateIfNewer is set.
	// Defaults to "last_update.date" if empty.
	ContentDateField string `json:"content_date_field"`

	// ContentDateFormat is the date format of ContentDateField, using the same
	// YYYY/MM/DD token style as -checkFormat.  Defaults to "YYYY-MM-DD".
	ContentDateFormat string `json:"content_date_format"`
}

func (c *LLMConfig) contentDateField() string {
	if c.ContentDateField != "" {
		return c.ContentDateField
	}
	return "last_update.date"
}

func (c *LLMConfig) contentDateFormat() string {
	if c.ContentDateFormat != "" {
		return c.ContentDateFormat
	}
	return "YYYY-MM-DD"
}

// LoadFMCConfig reads ~/.fmc/config.json.  Returns an empty config (no error)
// when the file does not exist, so callers can check fields individually.
func LoadFMCConfig() (FMCConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return FMCConfig{}, fmt.Errorf("could not determine home directory: %w", err)
	}
	path := filepath.Join(home, ".fmc", "config.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return FMCConfig{}, nil
	}
	if err != nil {
		return FMCConfig{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg FMCConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FMCConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}
