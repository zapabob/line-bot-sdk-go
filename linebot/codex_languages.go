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
	"regexp"
	"strings"
	"sync"
)

// LanguageDefinition defines a programming language
type LanguageDefinition struct {
	// Name is the language identifier
	Name CodexLanguage `json:"name"`

	// DisplayName is the human-readable name
	DisplayName string `json:"display_name"`

	// FileExtensions are common file extensions for this language
	FileExtensions []string `json:"file_extensions"`

	// Keywords are language-specific keywords for detection
	Keywords []string `json:"keywords"`

	// Patterns are regex patterns for language detection
	Patterns []string `json:"patterns"`

	// CommentStyle defines how comments are written
	CommentStyle CommentStyle `json:"comment_style"`

	// SyntaxRules defines language-specific syntax rules
	SyntaxRules SyntaxRules `json:"syntax_rules"`

	// CodeGenerationHints provides hints for code generation
	CodeGenerationHints CodeGenerationHints `json:"code_generation_hints"`
}

// CommentStyle defines comment styles for a language
type CommentStyle struct {
	SingleLine string   `json:"single_line"` // e.g., "//", "#"
	MultiLine  []string `json:"multi_line"`  // e.g., ["/*", "*/"]
	DocComment string   `json:"doc_comment,omitempty"` // e.g., "///", "/**"
}

// SyntaxRules defines syntax rules for a language
type SyntaxRules struct {
	CaseSensitive    bool     `json:"case_sensitive"`
	IndentationStyle string   `json:"indentation_style"` // "spaces", "tabs", "both"
	IndentationSize  int      `json:"indentation_size"`
	LineEnding        string   `json:"line_ending"` // "lf", "crlf", "cr"
	MaxLineLength     int      `json:"max_line_length,omitempty"`
	ReservedWords     []string `json:"reserved_words,omitempty"`
}

// CodeGenerationHints provides hints for AI code generation
type CodeGenerationHints struct {
	PreferredStyle      string   `json:"preferred_style"` // "functional", "oop", "procedural"
	CommonPatterns      []string `json:"common_patterns"`
	BestPractices       []string `json:"best_practices"`
	FrameworkSuggestions []string `json:"framework_suggestions,omitempty"`
	LinterRules         []string `json:"linter_rules,omitempty"`
}

// LanguageRegistry manages custom language definitions
type LanguageRegistry struct {
	languages map[CodexLanguage]*LanguageDefinition
	mu        sync.RWMutex
}

// NewLanguageRegistry creates a new language registry
func NewLanguageRegistry() *LanguageRegistry {
	registry := &LanguageRegistry{
		languages: make(map[CodexLanguage]*LanguageDefinition),
	}

	// Register built-in languages
	registry.registerBuiltInLanguages()

	return registry
}

// RegisterLanguage registers a custom language definition
func (lr *LanguageRegistry) RegisterLanguage(lang *LanguageDefinition) error {
	if lang == nil {
		return fmt.Errorf("language definition cannot be nil")
	}

	if lang.Name == "" {
		return fmt.Errorf("language name is required")
	}

	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.languages[lang.Name] = lang
	return nil
}

// GetLanguage returns a language definition
func (lr *LanguageRegistry) GetLanguage(lang CodexLanguage) (*LanguageDefinition, error) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	definition, exists := lr.languages[lang]
	if !exists {
		return nil, fmt.Errorf("language %s not registered", lang)
	}

	return definition, nil
}

// DetectLanguage detects the programming language from code
func (lr *LanguageRegistry) DetectLanguage(code string) CodexLanguage {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	codeLower := strings.ToLower(code)
	bestMatch := LanguageUnknown
	maxScore := 0

	for lang, definition := range lr.languages {
		score := 0

		// Check keywords
		for _, keyword := range definition.Keywords {
			if strings.Contains(codeLower, strings.ToLower(keyword)) {
				score += 2
			}
		}

		// Check patterns
		for _, pattern := range definition.Patterns {
			matched, _ := regexp.MatchString(pattern, code)
			if matched {
				score += 3
			}
		}

		// Check file extensions (if provided in context)
		// This would require additional context, so we skip it here

		if score > maxScore {
			maxScore = score
			bestMatch = lang
		}
	}

	if maxScore == 0 {
		return LanguageUnknown
	}

	return bestMatch
}

// ListLanguages returns all registered languages
func (lr *LanguageRegistry) ListLanguages() []CodexLanguage {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	languages := make([]CodexLanguage, 0, len(lr.languages))
	for lang := range lr.languages {
		languages = append(languages, lang)
	}

	return languages
}

// registerBuiltInLanguages registers built-in language definitions
func (lr *LanguageRegistry) registerBuiltInLanguages() {
	// Go
	lr.RegisterLanguage(&LanguageDefinition{
		Name:        LanguageGo,
		DisplayName: "Go",
		FileExtensions: []string{".go"},
		Keywords: []string{"package", "func", "import", "var", "const", "type", "interface", "struct"},
		Patterns: []string{`package\s+\w+`, `func\s+\w+\s*\(`},
		CommentStyle: CommentStyle{
			SingleLine: "//",
			MultiLine:  []string{"/*", "*/"},
			DocComment: "//",
		},
		SyntaxRules: SyntaxRules{
			CaseSensitive:    true,
			IndentationStyle: "tabs",
			IndentationSize:  0, // tabs
			LineEnding:        "lf",
			MaxLineLength:     100,
		},
		CodeGenerationHints: CodeGenerationHints{
			PreferredStyle: "functional",
			CommonPatterns: []string{"error handling", "interfaces", "goroutines"},
			BestPractices:  []string{"use interfaces", "handle errors explicitly", "follow Go conventions"},
		},
	})

	// Python
	lr.RegisterLanguage(&LanguageDefinition{
		Name:        LanguagePython,
		DisplayName: "Python",
		FileExtensions: []string{".py", ".pyw"},
		Keywords: []string{"def", "import", "class", "if", "elif", "else", "for", "while", "try", "except"},
		Patterns: []string{`def\s+\w+\s*\(`, `import\s+\w+`, `class\s+\w+`},
		CommentStyle: CommentStyle{
			SingleLine: "#",
			MultiLine:  []string{`"""`, `"""`},
			DocComment: `"""`,
		},
		SyntaxRules: SyntaxRules{
			CaseSensitive:    true,
			IndentationStyle: "spaces",
			IndentationSize:  4,
			LineEnding:        "lf",
			MaxLineLength:     88, // Black formatter default
		},
		CodeGenerationHints: CodeGenerationHints{
			PreferredStyle: "functional",
			CommonPatterns: []string{"list comprehensions", "decorators", "context managers"},
			BestPractices:  []string{"PEP 8", "type hints", "docstrings"},
		},
	})

	// JavaScript
	lr.RegisterLanguage(&LanguageDefinition{
		Name:        LanguageJavaScript,
		DisplayName: "JavaScript",
		FileExtensions: []string{".js", ".mjs", ".jsx"},
		Keywords: []string{"function", "const", "let", "var", "class", "async", "await", "export", "import"},
		Patterns: []string{`function\s+\w+`, `const\s+\w+\s*=`, `class\s+\w+`},
		CommentStyle: CommentStyle{
			SingleLine: "//",
			MultiLine:  []string{"/*", "*/"},
			DocComment: "/**",
		},
		SyntaxRules: SyntaxRules{
			CaseSensitive:    true,
			IndentationStyle: "spaces",
			IndentationSize:  2,
			LineEnding:        "lf",
		},
		CodeGenerationHints: CodeGenerationHints{
			PreferredStyle: "functional",
			CommonPatterns: []string{"arrow functions", "promises", "async/await"},
			BestPractices:  []string{"ES6+", "use strict", "avoid var"},
		},
	})

	// Add more built-in languages as needed...
}

// AnalyzeCode performs advanced code analysis
func (lr *LanguageRegistry) AnalyzeCode(ctx context.Context, code string, language CodexLanguage) (*CodeAnalysisResult, error) {
	definition, err := lr.GetLanguage(language)
	if err != nil {
		return nil, err
	}

	result := &CodeAnalysisResult{
		SecurityIssues:    make([]SecurityIssue, 0),
		PerformanceIssues: make([]PerformanceIssue, 0),
		Suggestions:       make([]string, 0),
		Metadata:          make(map[string]interface{}),
	}

	// Calculate complexity
	result.Complexity = lr.calculateComplexity(code, definition)

	// Detect security issues
	result.SecurityIssues = lr.detectSecurityIssues(code, definition)

	// Detect performance issues
	result.PerformanceIssues = lr.detectPerformanceIssues(code, definition)

	// Calculate code quality metrics
	result.CodeQuality = lr.calculateCodeQuality(code, definition)

	// Generate suggestions
	result.Suggestions = lr.generateSuggestions(code, definition)

	return result, nil
}

// calculateComplexity calculates cyclomatic complexity
func (lr *LanguageRegistry) calculateComplexity(code string, definition *LanguageDefinition) int {
	complexity := 1 // Base complexity

	// Count decision points based on language patterns
	decisionPatterns := []string{
		"if", "else", "elif", "switch", "case",
		"for", "while", "do", "catch", "except",
		"&&", "||", "?", "??",
	}

	for _, pattern := range decisionPatterns {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(pattern) + `\b`)
		matches := re.FindAllString(code, -1)
		complexity += len(matches)
	}

	return complexity
}

// detectSecurityIssues detects security vulnerabilities
func (lr *LanguageRegistry) detectSecurityIssues(code string, definition *LanguageDefinition) []SecurityIssue {
	issues := make([]SecurityIssue, 0)

	// SQL Injection detection
	if matched, _ := regexp.MatchString(`(?i)(query|execute|exec)\s*\([^)]*\+`, code); matched {
		issues = append(issues, SecurityIssue{
			Severity:    "high",
			Type:        "sql_injection",
			Description: "Potential SQL injection vulnerability detected",
			Fix:         "Use parameterized queries or prepared statements",
		})
	}

	// XSS detection
	if matched, _ := regexp.MatchString(`(?i)(innerHTML|outerHTML|document\.write)\s*=`, code); matched {
		issues = append(issues, SecurityIssue{
			Severity:    "medium",
			Type:        "xss",
			Description: "Potential XSS vulnerability detected",
			Fix:         "Use textContent or sanitize input",
		})
	}

	// Hardcoded secrets
	if matched, _ := regexp.MatchString(`(?i)(password|secret|api[_-]?key|token)\s*=\s*["'][^"']+["']`, code); matched {
		issues = append(issues, SecurityIssue{
			Severity:    "critical",
			Type:        "hardcoded_secret",
			Description: "Hardcoded secret detected",
			Fix:         "Use environment variables or secure configuration",
		})
	}

	return issues
}

// detectPerformanceIssues detects performance problems
func (lr *LanguageRegistry) detectPerformanceIssues(code string, definition *LanguageDefinition) []PerformanceIssue {
	issues := make([]PerformanceIssue, 0)

	// N+1 query pattern
	if matched, _ := regexp.MatchString(`for\s+.*\s+in\s+.*:\s*\n\s*.*\.(get|find|query)`, code); matched {
		issues = append(issues, PerformanceIssue{
			Severity:    "medium",
			Type:        "n_plus_one",
			Description: "Potential N+1 query problem detected",
			Impact:      "May cause significant performance degradation",
			Fix:         "Use eager loading or batch queries",
		})
	}

	// Inefficient string concatenation in loops
	if matched, _ := regexp.MatchString(`for\s+.*:\s*\n\s*.*\s*\+=\s*`, code); matched {
		issues = append(issues, PerformanceIssue{
			Severity:    "low",
			Type:        "inefficient_string_concat",
			Description: "Inefficient string concatenation in loop",
			Impact:      "May cause memory allocation overhead",
			Fix:         "Use string builder or join method",
		})
	}

	return issues
}

// calculateCodeQuality calculates code quality metrics
func (lr *LanguageRegistry) calculateCodeQuality(code string, definition *LanguageDefinition) CodeQualityMetrics {
	lines := strings.Split(code, "\n")
	totalLines := len(lines)
	codeLines := 0
	commentLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, definition.CommentStyle.SingleLine) {
			commentLines++
		} else {
			codeLines++
		}
	}

	maintainabilityIndex := 100.0
	if totalLines > 0 {
		complexity := lr.calculateComplexity(code, definition)
		maintainabilityIndex = 100.0 - float64(complexity)*5.0
		if maintainabilityIndex < 0 {
			maintainabilityIndex = 0
		}
	}

	return CodeQualityMetrics{
		MaintainabilityIndex: maintainabilityIndex,
		CyclomaticComplexity: lr.calculateComplexity(code, definition),
		CodeDuplication:      0.0, // Would require more sophisticated analysis
		DocumentationCoverage: float64(commentLines) / float64(codeLines+commentLines) * 100.0,
	}
}

// generateSuggestions generates code improvement suggestions
func (lr *LanguageRegistry) generateSuggestions(code string, definition *LanguageDefinition) []string {
	suggestions := make([]string, 0)

	// Check for missing error handling
	if matched, _ := regexp.MatchString(`\w+\([^)]*\)`, code); matched {
		if !strings.Contains(code, "error") && !strings.Contains(code, "catch") && !strings.Contains(code, "except") {
			suggestions = append(suggestions, "Consider adding error handling")
		}
	}

	// Check for long functions
	lines := strings.Split(code, "\n")
	if len(lines) > 50 {
		suggestions = append(suggestions, "Function is quite long, consider breaking it into smaller functions")
	}

	// Check for magic numbers
	if matched, _ := regexp.MatchString(`\b\d{3,}\b`, code); matched {
		suggestions = append(suggestions, "Consider replacing magic numbers with named constants")
	}

	return suggestions
}
