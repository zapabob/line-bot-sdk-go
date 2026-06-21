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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// CodexMode represents the mode of Codex operation
type CodexMode string

const (
	// CodexModeGenerate generates new code
	CodexModeGenerate CodexMode = "generate"
	// CodexModeReview reviews existing code
	CodexModeReview CodexMode = "review"
	// CodexModeFix fixes bugs in code
	CodexModeFix CodexMode = "fix"
	// CodexModeExplain explains code functionality
	CodexModeExplain CodexMode = "explain"
	// CodexModeRefactor refactors code for better quality
	CodexModeRefactor CodexMode = "refactor"
)

// CodexLanguage represents supported programming languages
type CodexLanguage string

const (
	LanguageGo         CodexLanguage = "go"
	LanguagePython     CodexLanguage = "python"
	LanguageJavaScript CodexLanguage = "javascript"
	LanguageTypeScript CodexLanguage = "typescript"
	LanguageJava       CodexLanguage = "java"
	LanguageRust       CodexLanguage = "rust"
	LanguageCPP        CodexLanguage = "cpp"
	LanguageC          CodexLanguage = "c"
	LanguageSQL        CodexLanguage = "sql"
	LanguageUnknown    CodexLanguage = "unknown"
)

// CodexRequest represents a request to Codex
type CodexRequest struct {
	// Mode specifies the operation mode
	Mode CodexMode `json:"mode"`
	// Language specifies the programming language
	Language CodexLanguage `json:"language"`
	// Code contains the input code (for review, fix, explain, refactor modes)
	Code string `json:"code,omitempty"`
	// Prompt contains the natural language description (for generate mode)
	Prompt string `json:"prompt,omitempty"`
	// Context provides additional context about the codebase
	Context string `json:"context,omitempty"`
	// Options contains additional configuration options
	Options CodexOptions `json:"options,omitempty"`
}

// CodexOptions contains configuration options for Codex
type CodexOptions struct {
	// Token limits
	MaxTokens   int `json:"max_tokens,omitempty"`   // Maximum tokens in response
	MaxInputTokens int `json:"max_input_tokens,omitempty"` // Maximum input tokens

	// Generation parameters
	Temperature     float64 `json:"temperature,omitempty"`      // Randomness (0.0-2.0)
	TopP           float64 `json:"top_p,omitempty"`            // Nucleus sampling (0.0-1.0)
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"` // Repetition penalty (-2.0-2.0)
	PresencePenalty  float64 `json:"presence_penalty,omitempty"`  // Topic diversity penalty (-2.0-2.0)

	// Code generation options
	IncludeTests      bool     `json:"include_tests,omitempty"`       // Include test code
	IncludeComments   bool     `json:"include_comments,omitempty"`    // Include comments
	StyleGuide        string   `json:"style_guide,omitempty"`         // Coding style preferences
	ProgrammingLanguage string `json:"programming_language,omitempty"` // Target language hint

	// Advanced options
	Seed             int64    `json:"seed,omitempty"`              // Random seed for reproducibility
	StopSequences   []string `json:"stop_sequences,omitempty"`    // Custom stop sequences
	ResponseFormat  string   `json:"response_format,omitempty"`   // Response format (json, text)
}

// CodexResponse represents a response from Codex
type CodexResponse struct {
	// Success indicates if the operation was successful
	Success bool `json:"success"`
	// Code contains the generated or modified code
	Code string `json:"code,omitempty"`
	// Explanation contains natural language explanation
	Explanation string `json:"explanation,omitempty"`
	// Suggestions contains improvement suggestions
	Suggestions []string `json:"suggestions,omitempty"`
	// Errors contains any errors that occurred
	Errors []CodexError `json:"errors,omitempty"`
	// Metadata contains additional information about the response
	Metadata CodexMetadata `json:"metadata"`
}

// CodexError represents an error in Codex processing
type CodexError struct {
	// Type categorizes the error
	Type string `json:"type"`
	// Message describes the error
	Message string `json:"message"`
	// Line indicates the line number where error occurred (if applicable)
	Line int `json:"line,omitempty"`
	// Column indicates the column number where error occurred (if applicable)
	Column int `json:"column,omitempty"`
}

// CodexMetadata contains metadata about the Codex response
type CodexMetadata struct {
	// ProcessingTime indicates how long the request took to process
	ProcessingTime time.Duration `json:"processing_time"`
	// Model indicates which AI model was used
	Model string `json:"model"`
	// TokensUsed indicates how many tokens were consumed
	TokensUsed int `json:"tokens_used"`
	// Timestamp indicates when the response was generated
	Timestamp time.Time `json:"timestamp"`
}

// CodexConfig contains configuration for Codex
type CodexConfig struct {
	// Authentication methods
	APIKey      string   // API key for direct API access
	ClientID    string   // OAuth 2.0 Client ID (ChatGPT Platform)
	ClientSecret string  // OAuth 2.0 Client Secret (ChatGPT Platform)
	RedirectURL string   // OAuth 2.0 Redirect URL
	UseOAuth    bool     // Use OAuth 2.0 instead of API key
	Scopes      []string // OAuth 2.0 scopes

	// API Configuration
	Model      string        // AI model (gpt-5-codex, gpt-4o, gpt-4-turbo)
	BaseURL    string        // API base URL
	APIVersion string        // API version (for Azure OpenAI)
	Timeout    time.Duration // Request timeout
	MaxRetries int           // Max retry attempts

	// Advanced Options
	DefaultOptions CodexOptions // Default generation options
	UseResponsesAPI bool       // Use new Responses API (recommended for GPT-5-Codex)
	EnableStreaming bool       // Enable streaming responses
}

// OAuthToken represents an OAuth 2.0 token
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// IsExpired checks if the token is expired
func (t *OAuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-time.Minute)) // 1 minute buffer
}

// CodexHandler handles Codex operations
type CodexHandler struct {
	config            CodexConfig
	client            *Client
	oauthToken        *OAuthToken
	httpClient        *http.Client
	tokenMutex        sync.RWMutex
	providerManager   *MultiAIProviderManager
	pluginManager     *PluginManager
	languageRegistry  *LanguageRegistry
}

// NewCodexHandler creates a new Codex handler
func NewCodexHandler(config CodexConfig, client *Client) (*CodexHandler, error) {
	// Validate authentication configuration
	if !config.UseOAuth && config.APIKey == "" {
		return nil, fmt.Errorf("either API key or OAuth configuration is required")
	}
	if config.UseOAuth {
		if config.ClientID == "" || config.ClientSecret == "" {
			return nil, fmt.Errorf("OAuth 2.0 requires ClientID and ClientSecret")
		}
		if config.RedirectURL == "" {
			config.RedirectURL = "http://localhost:8080/oauth/callback"
		}
		if len(config.Scopes) == 0 {
			config.Scopes = []string{"openid", "profile", "email", "model.request"}
		}
	}

	// Set default values
	if config.Model == "" {
		config.Model = "gpt-5-codex" // Latest model as default
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second // Longer timeout for GPT-5
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-15-preview"
	}

	// Set default options for GPT-5-Codex
	if config.DefaultOptions.MaxTokens == 0 {
		config.DefaultOptions.MaxTokens = 4000 // GPT-5-Codex supports higher limits
	}
	if config.DefaultOptions.Temperature == 0 {
		config.DefaultOptions.Temperature = 0.1 // Lower temperature for code generation
	}
	if config.DefaultOptions.TopP == 0 {
		config.DefaultOptions.TopP = 0.9
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Initialize managers
	providerManager := NewMultiAIProviderManager()
	pluginManager := NewPluginManager()
	languageRegistry := NewLanguageRegistry()

	return &CodexHandler{
		config:           config,
		client:           client,
		httpClient:       httpClient,
		tokenMutex:       sync.RWMutex{},
		providerManager:  providerManager,
		pluginManager:    pluginManager,
		languageRegistry: languageRegistry,
	}, nil
}

// ProcessRequest processes a Codex request
func (h *CodexHandler) ProcessRequest(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	startTime := time.Now()

	// Validate request
	if err := h.validateRequest(req); err != nil {
		return &CodexResponse{
			Success: false,
			Errors: []CodexError{{
				Type:    "validation_error",
				Message: err.Error(),
			}},
			Metadata: CodexMetadata{
				ProcessingTime: time.Since(startTime),
				Model:          h.config.Model,
				TokensUsed:     0,
				Timestamp:      time.Now(),
			},
		}, nil
	}

	// Execute pre-processor plugins
	processedReq, err := h.pluginManager.ExecutePreProcessors(ctx, &req)
	if err != nil {
		return &CodexResponse{
			Success: false,
			Errors: []CodexError{{
				Type:    "preprocessing_error",
				Message: err.Error(),
			}},
			Metadata: CodexMetadata{
				ProcessingTime: time.Since(startTime),
				Model:          h.config.Model,
				TokensUsed:     0,
				Timestamp:      time.Now(),
			},
		}, nil
	}

	// Perform advanced code analysis if code is provided
	var analysisResults []*CodeAnalysisResult
	if processedReq.Code != "" {
		analysisResults, err = h.pluginManager.ExecuteAnalyzers(ctx, processedReq.Code, processedReq.Language)
		if err != nil {
			// Log but don't fail the request
			// Analysis is optional
		}
	}

	// Process based on mode
	response, err := h.processByMode(ctx, *processedReq)
	if err != nil {
		return &CodexResponse{
			Success: false,
			Errors: []CodexError{{
				Type:    "processing_error",
				Message: err.Error(),
			}},
			Metadata: CodexMetadata{
				ProcessingTime: time.Since(startTime),
				Model:          h.config.Model,
				TokensUsed:     0,
				Timestamp:      time.Now(),
			},
		}, nil
	}

	// Execute post-processor plugins
	processedResp, err := h.pluginManager.ExecutePostProcessors(ctx, response)
	if err != nil {
		return &CodexResponse{
			Success: false,
			Errors: []CodexError{{
				Type:    "postprocessing_error",
				Message: err.Error(),
			}},
			Metadata: CodexMetadata{
				ProcessingTime: time.Since(startTime),
				Model:          h.config.Model,
				TokensUsed:     0,
				Timestamp:      time.Now(),
			},
		}, nil
	}

	// Add analysis results to response if available
	if len(analysisResults) > 0 {
		// Merge analysis results into suggestions
		for _, analysis := range analysisResults {
			processedResp.Suggestions = append(processedResp.Suggestions, analysis.Suggestions...)
		}
	}

	processedResp.Metadata.ProcessingTime = time.Since(startTime)
	processedResp.Metadata.Model = h.config.Model
	processedResp.Metadata.Timestamp = time.Now()
	processedResp.Success = true

	return processedResp, nil
}

// validateRequest validates a Codex request
func (h *CodexHandler) validateRequest(req CodexRequest) error {
	if req.Mode == "" {
		return fmt.Errorf("mode is required")
	}

	switch req.Mode {
	case CodexModeGenerate:
		if req.Prompt == "" {
			return fmt.Errorf("prompt is required for generate mode")
		}
	case CodexModeReview, CodexModeFix, CodexModeExplain, CodexModeRefactor:
		if req.Code == "" {
			return fmt.Errorf("code is required for %s mode", req.Mode)
		}
	default:
		return fmt.Errorf("invalid mode: %s", req.Mode)
	}

	if req.Language == "" {
		req.Language = h.detectLanguage(req.Code)
	}

	return nil
}

// detectLanguage attempts to detect the programming language from code
func (h *CodexHandler) detectLanguage(code string) CodexLanguage {
	code = strings.ToLower(code)

	// Simple language detection based on keywords and syntax
	if strings.Contains(code, "package ") && strings.Contains(code, "func ") {
		return LanguageGo
	}
	if strings.Contains(code, "def ") && strings.Contains(code, "import ") && strings.Contains(code, ":") {
		return LanguagePython
	}
	if strings.Contains(code, "function") && strings.Contains(code, "var ") && strings.Contains(code, "console.log") {
		return LanguageJavaScript
	}
	if strings.Contains(code, "interface") && strings.Contains(code, "class ") && strings.Contains(code, "public ") {
		return LanguageJava
	}
	if strings.Contains(code, "fn ") && strings.Contains(code, "let ") && strings.Contains(code, "use ") {
		return LanguageRust
	}
	if strings.Contains(code, "SELECT") && strings.Contains(code, "FROM") && strings.Contains(code, "WHERE") {
		return LanguageSQL
	}

	return LanguageUnknown
}

// GetAuthorizationURL generates OAuth 2.0 authorization URL
func (h *CodexHandler) GetAuthorizationURL(state string) (string, error) {
	if !h.config.UseOAuth {
		return "", fmt.Errorf("OAuth is not enabled")
	}

	baseURL := "https://auth.openai.com/authorize"
	params := url.Values{}
	params.Add("client_id", h.config.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", h.config.RedirectURL)
	params.Add("scope", strings.Join(h.config.Scopes, " "))
	params.Add("state", state)

	authURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	return authURL, nil
}

// ExchangeCodeForToken exchanges authorization code for access token
func (h *CodexHandler) ExchangeCodeForToken(ctx context.Context, code string) (*OAuthToken, error) {
	if !h.config.UseOAuth {
		return nil, fmt.Errorf("OAuth is not enabled")
	}

	tokenURL := "https://auth.openai.com/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", h.config.RedirectURL)
	data.Set("client_id", h.config.ClientID)
	data.Set("client_secret", h.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	h.tokenMutex.Lock()
	h.oauthToken = &token
	h.tokenMutex.Unlock()

	return &token, nil
}

// RefreshToken refreshes the OAuth 2.0 access token
func (h *CodexHandler) RefreshToken(ctx context.Context) (*OAuthToken, error) {
	h.tokenMutex.RLock()
	token := h.oauthToken
	h.tokenMutex.RUnlock()

	if token == nil || token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	tokenURL := "https://auth.openai.com/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", h.config.ClientID)
	data.Set("client_secret", h.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var newToken OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	newToken.ExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)

	h.tokenMutex.Lock()
	h.oauthToken = &newToken
	h.tokenMutex.Unlock()

	return &newToken, nil
}

// getValidToken returns a valid access token, refreshing if necessary
func (h *CodexHandler) getValidToken(ctx context.Context) (string, error) {
	if !h.config.UseOAuth {
		return "", fmt.Errorf("OAuth is not enabled")
	}

	h.tokenMutex.RLock()
	token := h.oauthToken
	h.tokenMutex.RUnlock()

	if token == nil {
		return "", fmt.Errorf("no OAuth token available, please authenticate first")
	}

	if token.IsExpired() {
		newToken, err := h.RefreshToken(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		token = newToken
	}

	return token.AccessToken, nil
}

// processByMode processes the request based on the specified mode
func (h *CodexHandler) processByMode(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Use Responses API for GPT-5-Codex
	if h.config.UseResponsesAPI && h.config.Model == "gpt-5-codex" {
		return h.processWithResponsesAPI(ctx, req)
	}

	// Fallback to legacy Chat Completions API
	return h.processWithChatAPI(ctx, req)
}

// processWithResponsesAPI processes requests using OpenAI's Responses API (recommended for GPT-5-Codex)
func (h *CodexHandler) processWithResponsesAPI(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Get authentication token
	var authToken string
	var err error

	if h.config.UseOAuth {
		authToken, err = h.getValidToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get OAuth token: %w", err)
		}
	} else {
		authToken = h.config.APIKey
	}

	// Build request payload
	requestBody := map[string]interface{}{
		"model": h.config.Model,
		"input": h.buildPromptForMode(req),
		"instruction": "You are an expert software engineer. Provide high-quality, well-documented code with best practices.",
		"max_output_tokens": req.Options.MaxTokens,
		"temperature": req.Options.Temperature,
		"top_p": req.Options.TopP,
		"stream": h.config.EnableStreaming,
	}

	if req.Options.FrequencyPenalty != 0 {
		requestBody["frequency_penalty"] = req.Options.FrequencyPenalty
	}
	if req.Options.PresencePenalty != 0 {
		requestBody["presence_penalty"] = req.Options.PresencePenalty
	}
	if req.Options.Seed != 0 {
		requestBody["seed"] = req.Options.Seed
	}
	if len(req.Options.StopSequences) > 0 {
		requestBody["stop"] = req.Options.StopSequences
	}

	// Make API request
	apiURL := fmt.Sprintf("%s/responses", h.config.BaseURL)
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if h.config.UseOAuth {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	} else {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	if h.config.APIVersion != "" {
		httpReq.Header.Set("OpenAI-Beta", fmt.Sprintf("responses=%s", h.config.APIVersion))
	}

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	// Extract response content
	output, ok := apiResponse["output"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: missing output field")
	}

	text, ok := output["text"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format: missing text field")
	}

	usage, _ := apiResponse["usage"].(map[string]interface{})
	tokensUsed := 0
	if usage != nil {
		if totalTokens, ok := usage["total_tokens"].(float64); ok {
			tokensUsed = int(totalTokens)
		}
	}

	return &CodexResponse{
		Code:        text,
		Explanation: h.generateExplanationForMode(req),
		Suggestions: h.generateSuggestionsForMode(req),
		Metadata: CodexMetadata{
			TokensUsed: tokensUsed,
		},
	}, nil
}

// processWithChatAPI processes requests using legacy Chat Completions API
func (h *CodexHandler) processWithChatAPI(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Get authentication token
	var authToken string
	var err error

	if h.config.UseOAuth {
		authToken, err = h.getValidToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get OAuth token: %w", err)
		}
	} else {
		authToken = h.config.APIKey
	}

	// Build messages for Chat Completions API
	systemMessage := "You are an expert software engineer. Provide high-quality, well-documented code following best practices."
	userMessage := h.buildPromptForMode(req)

	messages := []map[string]interface{}{
		{"role": "system", "content": systemMessage},
		{"role": "user", "content": userMessage},
	}

	requestBody := map[string]interface{}{
		"model": h.config.Model,
		"messages": messages,
		"max_tokens": req.Options.MaxTokens,
		"temperature": req.Options.Temperature,
		"top_p": req.Options.TopP,
		"stream": h.config.EnableStreaming,
	}

	if req.Options.FrequencyPenalty != 0 {
		requestBody["frequency_penalty"] = req.Options.FrequencyPenalty
	}
	if req.Options.PresencePenalty != 0 {
		requestBody["presence_penalty"] = req.Options.PresencePenalty
	}
	if req.Options.Seed != 0 {
		requestBody["seed"] = req.Options.Seed
	}
	if len(req.Options.StopSequences) > 0 {
		requestBody["stop"] = req.Options.StopSequences
	}

	// Make API request
	apiURL := fmt.Sprintf("%s/chat/completions", h.config.BaseURL)
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if h.config.UseOAuth {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	} else {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from API")
	}

	return &CodexResponse{
		Code:        apiResponse.Choices[0].Message.Content,
		Explanation: h.generateExplanationForMode(req),
		Suggestions: h.generateSuggestionsForMode(req),
		Metadata: CodexMetadata{
			TokensUsed: apiResponse.Usage.TotalTokens,
		},
	}, nil
}

// buildPromptForMode builds the appropriate prompt based on the request mode
func (h *CodexHandler) buildPromptForMode(req CodexRequest) string {
	basePrompt := fmt.Sprintf("Language: %s\n", req.Language)

	switch req.Mode {
	case CodexModeGenerate:
		prompt := fmt.Sprintf("Generate %s code for the following requirement:\n\n%s", req.Language, req.Prompt)
		if req.Context != "" {
			prompt += fmt.Sprintf("\n\nAdditional context:\n%s", req.Context)
		}
		if req.Options.IncludeTests {
			prompt += "\n\nPlease include comprehensive unit tests."
		}
		if req.Options.IncludeComments {
			prompt += "\n\nPlease include detailed comments and documentation."
		}
		return basePrompt + prompt

	case CodexModeReview:
		return basePrompt + fmt.Sprintf("Please review the following %s code and provide feedback on code quality, potential bugs, and improvements:\n\n```%s\n%s\n```", req.Language, req.Language, req.Code)

	case CodexModeFix:
		return basePrompt + fmt.Sprintf("Please fix any bugs in the following %s code and provide the corrected version:\n\n```%s\n%s\n```", req.Language, req.Language, req.Code)

	case CodexModeExplain:
		return basePrompt + fmt.Sprintf("Please explain what the following %s code does, including the logic flow and any important concepts:\n\n```%s\n%s\n```", req.Language, req.Language, req.Code)

	case CodexModeRefactor:
		return basePrompt + fmt.Sprintf("Please refactor the following %s code for better readability, maintainability, and performance:\n\n```%s\n%s\n```", req.Language, req.Language, req.Code)

	default:
		return basePrompt + fmt.Sprintf("Process the following request: %s", req.Prompt)
	}
}

// generateExplanationForMode generates appropriate explanation text for each mode
func (h *CodexHandler) generateExplanationForMode(req CodexRequest) string {
	switch req.Mode {
	case CodexModeGenerate:
		return fmt.Sprintf("Generated %s code based on your requirements", req.Language)
	case CodexModeReview:
		return fmt.Sprintf("Code review completed for %s code", req.Language)
	case CodexModeFix:
		return fmt.Sprintf("Fixed bugs in %s code", req.Language)
	case CodexModeExplain:
		return fmt.Sprintf("Code explanation provided for %s code", req.Language)
	case CodexModeRefactor:
		return fmt.Sprintf("Refactored %s code for better quality", req.Language)
	default:
		return "Request processed successfully"
	}
}

// generateSuggestionsForMode generates appropriate suggestions for each mode
func (h *CodexHandler) generateSuggestionsForMode(req CodexRequest) []string {
	switch req.Mode {
	case CodexModeGenerate:
		return []string{
			"Review the generated code for correctness",
			"Add proper error handling where needed",
			"Consider adding input validation",
			"Run the code to verify it works as expected",
		}
	case CodexModeReview:
		return []string{
			"Consider the suggestions provided",
			"Run linters to check code quality",
			"Add unit tests for critical functions",
			"Review for security vulnerabilities",
		}
	case CodexModeFix:
		return []string{
			"Test the fixed code thoroughly",
			"Run existing test suites",
			"Check for edge cases that might still cause issues",
			"Consider adding regression tests",
		}
	case CodexModeExplain:
		return []string{
			"Use this explanation to understand the code better",
			"Consider documenting complex parts",
			"Ask for clarification on unclear sections",
		}
	case CodexModeRefactor:
		return []string{
			"Test the refactored code to ensure functionality is preserved",
			"Run performance benchmarks if applicable",
			"Update any related documentation",
			"Consider peer review of the changes",
		}
	default:
		return []string{"Request completed successfully"}
	}
}

// generateCode generates new code based on a prompt (legacy method - now uses APIs)
func (h *CodexHandler) generateCode(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	return h.processByMode(ctx, req)
}

// reviewCode reviews existing code
func (h *CodexHandler) reviewCode(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Placeholder implementation
	return &CodexResponse{
		Code:        req.Code, // return original code unchanged
		Explanation: fmt.Sprintf("Code review for %s code", req.Language),
		Suggestions: []string{
			"Consider adding more comments",
			"Check for potential race conditions",
			"Verify error handling is comprehensive",
		},
		Metadata: CodexMetadata{
			TokensUsed: 100, // placeholder
		},
	}, nil
}

// fixCode fixes bugs in code
func (h *CodexHandler) fixCode(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Placeholder implementation
	return &CodexResponse{
		Code:        req.Code, // return original code (would be fixed)
		Explanation: fmt.Sprintf("Fixed code for %s", req.Language),
		Suggestions: []string{
			"Test the fixes thoroughly",
			"Run linter to ensure code quality",
		},
		Metadata: CodexMetadata{
			TokensUsed: 120, // placeholder
		},
	}, nil
}

// explainCode explains code functionality
func (h *CodexHandler) explainCode(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Placeholder implementation
	return &CodexResponse{
		Code:        req.Code,
		Explanation: fmt.Sprintf("This %s code implements the following functionality...", req.Language),
		Suggestions: []string{
			"Consider documenting complex algorithms",
			"Add usage examples",
		},
		Metadata: CodexMetadata{
			TokensUsed: 80, // placeholder
		},
	}, nil
}

// refactorCode refactors code for better quality
func (h *CodexHandler) refactorCode(ctx context.Context, req CodexRequest) (*CodexResponse, error) {
	// Placeholder implementation
	return &CodexResponse{
		Code:        req.Code, // return refactored code
		Explanation: fmt.Sprintf("Refactored %s code for better maintainability", req.Language),
		Suggestions: []string{
			"Run tests to ensure functionality is preserved",
			"Consider performance implications of changes",
		},
		Metadata: CodexMetadata{
			TokensUsed: 110, // placeholder
		},
	}, nil
}
