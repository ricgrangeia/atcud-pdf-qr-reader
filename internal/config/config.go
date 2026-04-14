// Package config loads application settings from environment variables.
package config

import "os"

// AppVersion, LastUpdated e Author são definidos aqui e expostos
// pelo endpoint GET /api/v1/version.
const (
	AppVersion  = "1.0.0"
	Author      = "Ricardo Grangeia"
	AuthorURL   = "https://ricardo.grangeia.pt"
)

// Config holds every setting the application needs at startup.
// All values come from environment variables so the image is config-agnostic.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// VLLMBaseURL is the base URL of the vLLM-compatible OpenAI endpoint.
	// Available for AI-assisted processing if needed in the future.
	VLLMBaseURL string

	// VLLMAPIKey is the bearer token for authenticating with the vLLM service.
	VLLMAPIKey string

	// VLLMModel is the model identifier to use when calling vLLM.
	VLLMModel string
}

// Load reads the configuration from environment variables.
// PORT defaults to 8080 when not set.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:        port,
		VLLMBaseURL: os.Getenv("VLLM_BASE_URL"),
		VLLMAPIKey:  os.Getenv("VLLM_API_KEY"),
		VLLMModel:   os.Getenv("VLLM_MODEL"),
	}
}
