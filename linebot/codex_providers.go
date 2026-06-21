// Copyright 2025 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"context"
	"fmt"
	"sync"
)

// AIProviderType represents the type of AI service provider
type AIProviderType string

const (
	// ProviderOpenAI represents OpenAI (GPT models)
	ProviderOpenAI AIProviderType = "openai"
	// ProviderAnthropic represents Anthropic (Claude models)
	ProviderAnthropic AIProviderType = "anthropic"
	// ProviderGoogle represents Google (Gemini models)
	ProviderGoogle AIProviderType = "google"
	// ProviderDeepSeek represents DeepSeek
	ProviderDeepSeek AIProviderType = "deepseek"
	// ProviderGrok represents xAI (Grok models)
	ProviderGrok AIProviderType = "grok"
)

// AIProviderConfig contains configuration for an AI provider
type AIProviderConfig struct {
	Type        AIProviderType `json:"type"`
	APIKey      string         `json:"api_key,omitempty"`
	BaseURL     string         `json:"base_url,omitempty"`
	Model       string         `json:"model,omitempty"`
	Enabled     bool           `json:"enabled"`
	Timeout     int            `json:"timeout,omitempty"` // seconds
	MaxRetries  int            `json:"max_retries,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
}

// AIProviderResponse represents a response from an AI provider
type AIProviderResponse struct {
	Content      string            `json:"content"`
	Model        string            `json:"model"`
	Provider     AIProviderType    `json:"provider"`
	TokensUsed   int               `json:"tokens_used"`
	FinishReason string            `json:"finish_reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AIProvider defines the interface for AI service providers
type AIProvider interface {
	// Type returns the provider type
	Type() AIProviderType

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool

	// ProcessRequest processes a codex request and returns a response
	ProcessRequest(ctx context.Context, req CodexRequest) (*AIProviderResponse, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig() error

	// GetCapabilities returns the capabilities of this provider
	GetCapabilities() ProviderCapabilities
}

// ProviderCapabilities describes what a provider can do
type ProviderCapabilities struct {
	SupportsCodeGeneration bool     `json:"supports_code_generation"`
	SupportsCodeReview      bool     `json:"supports_code_review"`
	SupportsCodeFix         bool     `json:"supports_code_fix"`
	SupportsCodeExplain     bool     `json:"supports_code_explain"`
	SupportsCodeRefactor    bool     `json:"supports_code_refactor"`
	SupportedLanguages      []string `json:"supported_languages"`
	MaxContextLength        int      `json:"max_context_length"`
	SupportsStreaming       bool     `json:"supports_streaming"`
}

// MultiAIProviderManager manages multiple AI providers
type MultiAIProviderManager struct {
	providers map[AIProviderType]AIProvider
	mu        sync.RWMutex
	defaultProvider AIProviderType
	fallbackOrder   []AIProviderType
}

// NewMultiAIProviderManager creates a new multi-AI provider manager
func NewMultiAIProviderManager() *MultiAIProviderManager {
	return &MultiAIProviderManager{
		providers: make(map[AIProviderType]AIProvider),
		defaultProvider: ProviderOpenAI,
		fallbackOrder: []AIProviderType{
			ProviderOpenAI,
			ProviderAnthropic,
			ProviderGoogle,
			ProviderDeepSeek,
			ProviderGrok,
		},
	}
}

// RegisterProvider registers an AI provider
func (m *MultiAIProviderManager) RegisterProvider(provider AIProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[provider.Type()] = provider
	return nil
}

// GetProvider returns a provider by type
func (m *MultiAIProviderManager) GetProvider(providerType AIProviderType) (AIProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerType)
	}

	if !provider.IsEnabled() {
		return nil, fmt.Errorf("provider %s is not enabled", providerType)
	}

	return provider, nil
}

// GetDefaultProvider returns the default provider
func (m *MultiAIProviderManager) GetDefaultProvider() (AIProvider, error) {
	return m.GetProvider(m.defaultProvider)
}

// SetDefaultProvider sets the default provider
func (m *MultiAIProviderManager) SetDefaultProvider(providerType AIProviderType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[providerType]; !exists {
		return fmt.Errorf("provider %s not registered", providerType)
	}

	m.defaultProvider = providerType
	return nil
}

// ProcessRequestWithProvider processes a request with a specific provider
func (m *MultiAIProviderManager) ProcessRequestWithProvider(
	ctx context.Context,
	req CodexRequest,
	providerType AIProviderType,
) (*AIProviderResponse, error) {
	provider, err := m.GetProvider(providerType)
	if err != nil {
		return nil, err
	}

	return provider.ProcessRequest(ctx, req)
}

// ProcessRequestWithFallback processes a request with fallback support
func (m *MultiAIProviderManager) ProcessRequestWithFallback(
	ctx context.Context,
	req CodexRequest,
	preferredProvider AIProviderType,
) (*AIProviderResponse, error) {
	// Try preferred provider first
	if provider, err := m.GetProvider(preferredProvider); err == nil {
		resp, err := provider.ProcessRequest(ctx, req)
		if err == nil {
			return resp, nil
		}
	}

	// Fallback to other providers in order
	for _, providerType := range m.fallbackOrder {
		if providerType == preferredProvider {
			continue // Already tried
		}

		if provider, err := m.GetProvider(providerType); err == nil {
			resp, err := provider.ProcessRequest(ctx, req)
			if err == nil {
				return resp, nil
			}
		}
	}

	return nil, fmt.Errorf("all providers failed or unavailable")
}

// ProcessRequestWithAll processes a request with all available providers
func (m *MultiAIProviderManager) ProcessRequestWithAll(
	ctx context.Context,
	req CodexRequest,
) (map[AIProviderType]*AIProviderResponse, error) {
	m.mu.RLock()
	providers := make(map[AIProviderType]AIProvider)
	for k, v := range m.providers {
		if v.IsEnabled() {
			providers[k] = v
		}
	}
	m.mu.RUnlock()

	results := make(map[AIProviderType]*AIProviderResponse)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for providerType, provider := range providers {
		wg.Add(1)
		go func(pt AIProviderType, p AIProvider) {
			defer wg.Done()

			resp, err := p.ProcessRequest(ctx, req)
			if err == nil {
				mu.Lock()
				results[pt] = resp
				mu.Unlock()
			}
		}(providerType, provider)
	}

	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("all providers failed")
	}

	return results, nil
}

// ListProviders returns a list of all registered providers
func (m *MultiAIProviderManager) ListProviders() []AIProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]AIProviderType, 0, len(m.providers))
	for providerType := range m.providers {
		providers = append(providers, providerType)
	}

	return providers
}

// ListEnabledProviders returns a list of enabled providers
func (m *MultiAIProviderManager) ListEnabledProviders() []AIProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]AIProviderType, 0)
	for providerType, provider := range m.providers {
		if provider.IsEnabled() {
			providers = append(providers, providerType)
		}
	}

	return providers
}

