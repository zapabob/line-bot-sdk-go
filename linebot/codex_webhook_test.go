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
	"testing"
	"time"
)

func TestNewCodexWebhookHandler(t *testing.T) {
	client, _ := New("test", "test")

	tests := []struct {
		name    string
		config  CodexConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CodexConfig{
				APIKey: "test-key",
				Model:  "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: CodexConfig{
				// Missing API key
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewCodexWebhookHandler(tt.config, client)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCodexWebhookHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && handler == nil {
				t.Error("NewCodexWebhookHandler() returned nil handler")
			}
		})
	}
}

func TestCodexWebhookHandler_isCodexCommand(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexWebhookHandler(CodexConfig{APIKey: "test"}, client)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "generate command",
			text:     "/generate go hello world",
			expected: true,
		},
		{
			name:     "review command",
			text:     "/review python",
			expected: true,
		},
		{
			name:     "fix command",
			text:     "/fix javascript",
			expected: true,
		},
		{
			name:     "explain command",
			text:     "/explain java",
			expected: true,
		},
		{
			name:     "refactor command",
			text:     "/refactor rust",
			expected: true,
		},
		{
			name:     "codex command",
			text:     "/codex help",
			expected: true,
		},
		{
			name:     "regular message",
			text:     "Hello, how are you?",
			expected: false,
		},
		{
			name:     "empty message",
			text:     "",
			expected: false,
		},
		{
			name:     "case insensitive",
			text:     "/GENERATE go test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isCodexCommand(tt.text)
			if result != tt.expected {
				t.Errorf("isCodexCommand(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestCodexWebhookHandler_parseCodexCommand(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexWebhookHandler(CodexConfig{APIKey: "test"}, client)

	tests := []struct {
		name        string
		text        string
		expectError bool
		expectedReq CodexRequest
	}{
		{
			name: "generate command",
			text: "/generate go create a hello world function",
			expectedReq: CodexRequest{
				Mode:     CodexModeGenerate,
				Language: LanguageGo,
				Prompt:   "create a hello world function",
			},
			expectError: false,
		},
		{
			name: "review command with code",
			text: "/review python\n\ndef hello():\n    print(\"Hello\")\n\nhello()",
			expectedReq: CodexRequest{
				Mode:     CodexModeReview,
				Language: LanguagePython,
				Code:     "\ndef hello():\n    print(\"Hello\")\n\nhello()",
			},
			expectError: false,
		},
		{
			name: "fix command",
			text: "/fix javascript\n\nfunction broken() {\n    console.log(\"fix me\")\n",
			expectedReq: CodexRequest{
				Mode:     CodexModeFix,
				Language: LanguageJavaScript,
				Code:     "\nfunction broken() {\n    console.log(\"fix me\")\n",
			},
			expectError: false,
		},
		{
			name: "invalid command format",
			text:     "not a command",
			expectError: true,
		},
		{
			name: "unsupported mode",
			text:     "/unsupported go test",
			expectError: true,
		},
		{
			name: "missing language",
			text:     "/generate",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := handler.parseCodexCommand(tt.text)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if req.Mode != tt.expectedReq.Mode {
				t.Errorf("mode = %v, expected %v", req.Mode, tt.expectedReq.Mode)
			}

			if req.Language != tt.expectedReq.Language {
				t.Errorf("language = %v, expected %v", req.Language, tt.expectedReq.Language)
			}

			if req.Prompt != tt.expectedReq.Prompt {
				t.Errorf("prompt = %q, expected %q", req.Prompt, tt.expectedReq.Prompt)
			}

			if req.Code != tt.expectedReq.Code {
				t.Errorf("code = %q, expected %q", req.Code, tt.expectedReq.Code)
			}
		})
	}
}

func TestCodexWebhookHandler_HandleWebhookEvents(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexWebhookHandler(CodexConfig{APIKey: "test"}, client)

	tests := []struct {
		name  string
		event *Event
	}{
		{
			name: "text message event",
			event: &Event{
				Type: EventTypeMessage,
				Message: &TextMessage{
					Text: "/generate go hello world",
				},
				ReplyToken: "test-token",
			},
		},
		{
			name: "follow event",
			event: &Event{
				Type:       EventTypeFollow,
				ReplyToken: "test-token",
			},
		},
		{
			name: "non-text message",
			event: &Event{
				Type:    EventTypeMessage,
				Message: &ImageMessage{}, // Non-text message
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test mainly ensures no panics occur
			// In a real scenario, you'd mock the bot client
			events := []*Event{tt.event}
			handler.HandleWebhookEvents(events, &Request{})
		})
	}
}

func TestCodexWebhookHandler_GetWebhookHandler(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexWebhookHandler(CodexConfig{APIKey: "test"}, client)

	webhookHandler, err := handler.GetWebhookHandler("test-secret")
	if err != nil {
		t.Errorf("GetWebhookHandler() error = %v", err)
		return
	}

	if webhookHandler == nil {
		t.Error("GetWebhookHandler() returned nil")
	}
}


