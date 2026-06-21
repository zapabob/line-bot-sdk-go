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
	"os"
	"path/filepath"
	"time"
)

// GeminiCLIMCPConfig contains configuration for Gemini CLI MCP server
type GeminiCLIMCPConfig struct {
	// Command to start Gemini CLI MCP server
	// Default: "gemini" (if installed via npm/gemini CLI)
	Command string

	// Args for the command
	Args []string

	// API Key (optional, uses Google Cloud auth if not provided)
	APIKey string

	// Model to use
	Model string

	// Enabled flag
	Enabled bool

	// Timeout for requests
	Timeout time.Duration
}

// NewGeminiCLIMCPClient creates a new Gemini CLI MCP client
func NewGeminiCLIMCPClient(config GeminiCLIMCPConfig) (MCPClient, error) {
	if config.Command == "" {
		// Try to find gemini CLI
		if path, err := findCommand("gemini"); err == nil {
			config.Command = path
		} else {
			// Try npm global installation
			config.Command = "npx"
			config.Args = []string{"-y", "@google/gemini-cli"}
		}
	}

	if config.Model == "" {
		config.Model = "gemini-2.0-flash"
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	mcpConfig := MCPServerConfig{
		Type:    MCPServerGeminiCLI,
		Command: config.Command,
		Args:    config.Args,
		APIKey:  config.APIKey,
		Enabled: config.Enabled,
		Timeout: config.Timeout,
	}

	return NewStdioMCPClient(mcpConfig), nil
}

// ClaudeCodeMCPConfig contains configuration for Claude Code MCP server
type ClaudeCodeMCPConfig struct {
	// Command to start Claude Code MCP server
	// Default: "claude" (if installed)
	Command string

	// Args for the command
	Args []string

	// API Key (Anthropic API key)
	APIKey string

	// Model to use
	Model string

	// Enabled flag
	Enabled bool

	// Timeout for requests
	Timeout time.Duration
}

// NewClaudeCodeMCPClient creates a new Claude Code MCP client
func NewClaudeCodeMCPClient(config ClaudeCodeMCPConfig) (MCPClient, error) {
	if config.Command == "" {
		// Try to find claude CLI
		if path, err := findCommand("claude"); err == nil {
			config.Command = path
		} else {
			// Try npm global installation
			config.Command = "npx"
			config.Args = []string{"-y", "@anthropic/claude-code"}
		}
	}

	if config.Model == "" {
		config.Model = "claude-3-5-sonnet-20241022"
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	mcpConfig := MCPServerConfig{
		Type:    MCPServerClaudeCode,
		Command: config.Command,
		Args:    config.Args,
		APIKey:  config.APIKey,
		Enabled: config.Enabled,
		Timeout: config.Timeout,
	}

	return NewStdioMCPClient(mcpConfig), nil
}

// RegisterMCPProviders registers MCP-based providers to the manager
func (h *CodexHandler) RegisterMCPProviders(ctx context.Context, geminiConfig *GeminiCLIMCPConfig, claudeConfig *ClaudeCodeMCPConfig) error {
	// Register Gemini CLI MCP
	if geminiConfig != nil && geminiConfig.Enabled {
		geminiClient, err := NewGeminiCLIMCPClient(*geminiConfig)
		if err != nil {
			return fmt.Errorf("failed to create Gemini CLI MCP client: %w", err)
		}

		if err := geminiClient.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to Gemini CLI MCP: %w", err)
		}

		adapter := NewMCPProviderAdapter(geminiClient, AIProviderConfig{
			Type:    ProviderGoogle,
			Model:   geminiConfig.Model,
			Enabled: true,
		})

		if err := h.providerManager.RegisterProvider(adapter); err != nil {
			return fmt.Errorf("failed to register Gemini CLI provider: %w", err)
		}
	}

	// Register Claude Code MCP
	if claudeConfig != nil && claudeConfig.Enabled {
		claudeClient, err := NewClaudeCodeMCPClient(*claudeConfig)
		if err != nil {
			return fmt.Errorf("failed to create Claude Code MCP client: %w", err)
		}

		if err := claudeClient.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to Claude Code MCP: %w", err)
		}

		adapter := NewMCPProviderAdapter(claudeClient, AIProviderConfig{
			Type:    ProviderAnthropic,
			Model:   claudeConfig.Model,
			Enabled: true,
		})

		if err := h.providerManager.RegisterProvider(adapter); err != nil {
			return fmt.Errorf("failed to register Claude Code provider: %w", err)
		}
	}

	return nil
}

// findCommand finds a command in PATH
func findCommand(name string) (string, error) {
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("command not found: %s", name)
}

