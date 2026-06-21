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
	"testing"
	"time"
)

func TestNewCodexHandler(t *testing.T) {
	tests := []struct {
		name    string
		config  CodexConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CodexConfig{
				APIKey:     "test-key",
				Model:      "gpt-4",
				BaseURL:    "https://api.openai.com/v1",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: CodexConfig{
				Model:      "gpt-4",
				BaseURL:    "https://api.openai.com/v1",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			wantErr: true,
		},
		{
			name: "empty config gets defaults",
			config: CodexConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := New("test", "test")
			handler, err := NewCodexHandler(tt.config, client)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCodexHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && handler == nil {
				t.Error("NewCodexHandler() returned nil handler")
			}
		})
	}
}

func TestCodexHandler_validateRequest(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexHandler(CodexConfig{APIKey: "test"}, client)

	tests := []struct {
		name    string
		req     CodexRequest
		wantErr bool
	}{
		{
			name: "valid generate request",
			req: CodexRequest{
				Mode:     CodexModeGenerate,
				Language: LanguageGo,
				Prompt:   "create a hello world function",
			},
			wantErr: false,
		},
		{
			name: "valid review request",
			req: CodexRequest{
				Mode:     CodexModeReview,
				Language: LanguageGo,
				Code:     "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
			},
			wantErr: false,
		},
		{
			name: "missing mode",
			req: CodexRequest{
				Language: LanguageGo,
				Code:     "some code",
			},
			wantErr: true,
		},
		{
			name: "generate without prompt",
			req: CodexRequest{
				Mode:     CodexModeGenerate,
				Language: LanguageGo,
			},
			wantErr: true,
		},
		{
			name: "review without code",
			req: CodexRequest{
				Mode:     CodexModeReview,
				Language: LanguageGo,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCodexHandler_detectLanguage(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexHandler(CodexConfig{APIKey: "test"}, client)

	tests := []struct {
		name     string
		code     string
		expected CodexLanguage
	}{
		{
			name:     "Go code",
			code:     "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
			expected: LanguageGo,
		},
		{
			name:     "Python code",
			code:     "def hello():\n    print(\"Hello\")\n\nif __name__ == \"__main__\":\n    hello()",
			expected: LanguagePython,
		},
		{
			name:     "JavaScript code",
			code:     "function hello() {\n    console.log(\"Hello\");\n}\n\nhello();",
			expected: LanguageJavaScript,
		},
		{
			name:     "Java code",
			code:     "public class Hello {\n    public static void main(String[] args) {\n        System.out.println(\"Hello\");\n    }\n}",
			expected: LanguageJava,
		},
		{
			name:     "Rust code",
			code:     "fn main() {\n    println!(\"Hello\");\n}",
			expected: LanguageRust,
		},
		{
			name:     "SQL code",
			code:     "SELECT * FROM users WHERE id = 1;",
			expected: LanguageSQL,
		},
		{
			name:     "unknown code",
			code:     "some random text that is not code",
			expected: LanguageUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.detectLanguage(tt.code)
			if result != tt.expected {
				t.Errorf("detectLanguage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCodexHandler_ProcessRequest(t *testing.T) {
	client, _ := New("test", "test")
	handler, _ := NewCodexHandler(CodexConfig{APIKey: "test"}, client)

	tests := []struct {
		name        string
		req         CodexRequest
		expectError bool
	}{
		{
			name: "generate code",
			req: CodexRequest{
				Mode:     CodexModeGenerate,
				Language: LanguageGo,
				Prompt:   "create a hello world function",
			},
			expectError: false,
		},
		{
			name: "review code",
			req: CodexRequest{
				Mode:     CodexModeReview,
				Language: LanguageGo,
				Code:     "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
			},
			expectError: false,
		},
		{
			name: "invalid request",
			req: CodexRequest{
				Mode:     CodexMode("invalid"),
				Language: LanguageGo,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := handler.ProcessRequest(ctx, tt.req)

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

			if resp == nil {
				t.Error("response is nil")
				return
			}

			if resp.Metadata.Timestamp.IsZero() {
				t.Error("timestamp should not be zero")
			}

			if resp.Metadata.Model == "" {
				t.Error("model should not be empty")
			}
		})
	}
}

func TestCodexModes(t *testing.T) {
	// Test that all modes are properly defined
	modes := []CodexMode{
		CodexModeGenerate,
		CodexModeReview,
		CodexModeFix,
		CodexModeExplain,
		CodexModeRefactor,
	}

	for _, mode := range modes {
		if mode == "" {
			t.Error("mode should not be empty")
		}
	}
}

func TestCodexLanguages(t *testing.T) {
	// Test that all languages are properly defined
	languages := []CodexLanguage{
		LanguageGo,
		LanguagePython,
		LanguageJavaScript,
		LanguageTypeScript,
		LanguageJava,
		LanguageRust,
		LanguageCPP,
		LanguageC,
		LanguageSQL,
		LanguageUnknown,
	}

	for _, lang := range languages {
		if lang == "" {
			t.Error("language should not be empty")
		}
	}
}


