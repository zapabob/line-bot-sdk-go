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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

// WebhookServer represents the Codex webhook server
type WebhookServer struct {
	bot          *linebot.Client
	codexHandler *linebot.CodexWebhookHandler
	server       *http.Server
}

// NewWebhookServer creates a new webhook server instance
func NewWebhookServer() (*WebhookServer, error) {
	// 環境変数から設定を読み込み
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	// 必須環境変数の検証
	if channelSecret == "" {
		return nil, ErrMissingEnvVar{Var: "LINE_CHANNEL_SECRET"}
	}
	if channelToken == "" {
		return nil, ErrMissingEnvVar{Var: "LINE_CHANNEL_ACCESS_TOKEN"}
	}
	if openaiAPIKey == "" {
		return nil, ErrMissingEnvVar{Var: "OPENAI_API_KEY"}
	}

	// LINE Bot クライアント作成
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		return nil, ErrBotCreation{Err: err}
	}

	// Codex設定
	codexConfig := createCodexConfig(openaiAPIKey)

	// Codex Handler作成（MCP統合用）
	codexHandlerCore, err := linebot.NewCodexHandler(codexConfig, bot)
	if err != nil {
		return nil, ErrCodexHandlerCreation{Err: err}
	}

	// MCPプロバイダーを登録（オプション、環境変数で無効化可能）
	if os.Getenv("DISABLE_MCP") != "true" {
		if err := registerMCPProviders(codexHandlerCore); err != nil {
			log.Printf("⚠️  Warning: Failed to register MCP providers: %v", err)
			log.Printf("ℹ️  Continuing without MCP providers...")
			log.Printf("💡  To disable MCP completely, set DISABLE_MCP=true")
		}
	} else {
		log.Printf("ℹ️  MCP providers are disabled (DISABLE_MCP=true)")
	}

	// Codex Webhook ハンドラー作成
	codexHandler, err := linebot.NewCodexWebhookHandler(codexConfig, bot)
	if err != nil {
		return nil, ErrCodexWebhookHandlerCreation{Err: err}
	}

	// Webhook ハンドラー取得
	webhookHandler, err := codexHandler.GetWebhookHandler(channelSecret)
	if err != nil {
		return nil, ErrWebhookHandlerCreation{Err: err}
	}

	// HTTPサーバー設定
	port := getEnvOrDefault("PORT", "8080")
	mux := http.NewServeMux()

	// LINE API Send Proxy
	lineProxy := createLineProxy(channelToken)
	mux.Handle("/v2/", lineProxy)

	// Hybrid Webhook Handler (Codex / Hermes)
	hybridHandler := createHybridWebhookHandler(channelSecret, webhookHandler)
	mux.Handle("/webhook", hybridHandler)

	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/", rootHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &WebhookServer{
		bot:          bot,
		codexHandler: codexHandler,
		server:       server,
	}, nil
}

// Start starts the webhook server
func (s *WebhookServer) Start() error {
	log.Printf("🚀 Codex Bot Server starting...")
	log.Printf("📝 Webhook URL: http://localhost%s/webhook", s.server.Addr)
	log.Printf("💡 Health check: http://localhost%s/health", s.server.Addr)
	log.Printf("✅ Server is ready to receive webhook events")

	// グレースフルシャットダウンの設定
	go s.handleShutdown()

	return s.server.ListenAndServe()
}

// handleShutdown handles graceful shutdown
func (s *WebhookServer) handleShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	} else {
		log.Printf("✅ Server exited gracefully")
	}
}

// createCodexConfig creates Codex configuration from environment variables
func createCodexConfig(apiKey string) linebot.CodexConfig {
	config := linebot.CodexConfig{
		APIKey:  apiKey,
		Model:   getEnvOrDefault("OPENAI_MODEL", "gpt-5-codex"),
		BaseURL: getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		Timeout: 60 * time.Second,
		DefaultOptions: linebot.CodexOptions{
			MaxTokens:        4000,
			Temperature:      0.1,
			TopP:             0.9,
			FrequencyPenalty: 0.0,
			PresencePenalty:  0.0,
			IncludeTests:     true,
			IncludeComments:  true,
		},
		UseResponsesAPI: getEnvOrDefault("USE_RESPONSES_API", "true") == "true",
	}

	// OAuth 2.0設定（オプション）
	if getEnvOrDefault("USE_OAUTH", "false") == "true" {
		config.UseOAuth = true
		config.ClientID = os.Getenv("CHATGPT_CLIENT_ID")
		config.ClientSecret = os.Getenv("CHATGPT_CLIENT_SECRET")
		config.RedirectURL = os.Getenv("CHATGPT_REDIRECT_URL")
		config.Scopes = []string{"openid", "profile", "email", "model.request"}
	}

	return config
}

// registerMCPProviders registers MCP providers if enabled
func registerMCPProviders(codexHandler *linebot.CodexHandler) error {
	ctx := context.Background()

	// Gemini CLI MCP設定（オプション）
	geminiEnabled := os.Getenv("GEMINI_MCP_ENABLED") == "true"
	claudeEnabled := os.Getenv("CLAUDE_MCP_ENABLED") == "true"

	if !geminiEnabled && !claudeEnabled {
		return nil // MCPが無効化されている場合は何もしない
	}

	var geminiConfig *linebot.GeminiCLIMCPConfig
	var claudeConfig *linebot.ClaudeCodeMCPConfig

	if geminiEnabled {
		geminiConfig = &linebot.GeminiCLIMCPConfig{
			Command: getEnvOrDefault("GEMINI_CLI_COMMAND", ""), // 自動検出
			Model:   getEnvOrDefault("GEMINI_MODEL", "gemini-2.0-flash"),
			Enabled: true,
			Timeout: 60 * time.Second,
		}
	}

	if claudeEnabled {
		claudeAPIKey := os.Getenv("ANTHROPIC_API_KEY")
		if claudeAPIKey == "" {
			log.Printf("⚠️  ANTHROPIC_API_KEY not set, skipping Claude Code MCP")
		} else {
			claudeConfig = &linebot.ClaudeCodeMCPConfig{
				Command: getEnvOrDefault("CLAUDE_CLI_COMMAND", ""), // 自動検出
				APIKey:  claudeAPIKey,
				Model:   getEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20241022"),
				Enabled: true,
				Timeout: 60 * time.Second,
			}
		}
	}

	// MCPプロバイダーを登録
	return codexHandler.RegisterMCPProviders(ctx, geminiConfig, claudeConfig)
}

// HTTP Handlers

// healthCheckHandler handles health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"codex-bot"}`))
}

// rootHandler handles root path requests
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Codex Bot Webhook Server\n\nEndpoints:\n  POST /webhook - LINE webhook endpoint\n  GET  /health  - Health check endpoint"))
}

// Helper functions

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Error types

type ErrMissingEnvVar struct {
	Var string
}

func (e ErrMissingEnvVar) Error() string {
	return fmt.Sprintf("missing required environment variable: %s", e.Var)
}

type ErrBotCreation struct {
	Err error
}

func (e ErrBotCreation) Error() string {
	return fmt.Sprintf("failed to create LINE bot client: %v", e.Err)
}

type ErrCodexHandlerCreation struct {
	Err error
}

func (e ErrCodexHandlerCreation) Error() string {
	return fmt.Sprintf("failed to create Codex handler: %v", e.Err)
}

type ErrCodexWebhookHandlerCreation struct {
	Err error
}

func (e ErrCodexWebhookHandlerCreation) Error() string {
	return fmt.Sprintf("failed to create Codex webhook handler: %v", e.Err)
}

type ErrWebhookHandlerCreation struct {
	Err error
}

func (e ErrWebhookHandlerCreation) Error() string {
	return fmt.Sprintf("failed to create webhook handler: %v", e.Err)
}

// ---------------------------------------------------------------------------
// Proxy and Forwarding Integrations
// ---------------------------------------------------------------------------

type webhookPayload struct {
	Events []struct {
		Type    string `json:"type"`
		Message struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"events"`
}

func createLineProxy(channelToken string) http.Handler {
	target, _ := url.Parse("https://api.line.me")
	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("Authorization", "Bearer "+channelToken)
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
	}
	return proxy
}

func createHybridWebhookHandler(channelSecret string, codexHandler http.Handler) http.Handler {
	forwardToHermes := os.Getenv("FORWARD_TO_HERMES") == "true"
	hermesURL := getEnvOrDefault("HERMES_WEBHOOK_URL", "http://localhost:8646/line/webhook")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Read body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		// Restore body for downstream handlers
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Check if it is a Codex command
		isCodexCommand := false
		var payload webhookPayload
		if err := json.Unmarshal(bodyBytes, &payload); err == nil {
			for _, event := range payload.Events {
				if event.Type == "message" && event.Message.Type == "text" {
					text := strings.TrimSpace(event.Message.Text)
					if strings.HasPrefix(text, "/generate") ||
						strings.HasPrefix(text, "/review") ||
						strings.HasPrefix(text, "/fix") ||
						strings.HasPrefix(text, "/explain") ||
						strings.HasPrefix(text, "/refactor") {
						isCodexCommand = true
						break
					}
				}
			}
		}

		if isCodexCommand {
			// Let Codex handle it
			codexHandler.ServeHTTP(w, r)
			return
		}

		if forwardToHermes {
			// Forward to Hermes
			log.Printf("Forwarding non-command Webhook to Hermes: %s", hermesURL)

			// Create a forward request
			req, err := http.NewRequest(http.MethodPost, hermesURL, bytes.NewReader(bodyBytes))
			if err != nil {
				log.Printf("Failed to create forward request: %v", err)
				http.Error(w, "Forward error", http.StatusInternalServerError)
				return
			}

			// Copy headers, especially signature
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Failed to forward request to Hermes: %v", err)
				http.Error(w, "Hermes offline", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			// Copy response back
			for k, vv := range resp.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
		}

		// Fallback if forwarding is disabled
		codexHandler.ServeHTTP(w, r)
	})
}
