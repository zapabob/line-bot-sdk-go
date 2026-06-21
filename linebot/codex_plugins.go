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

// PluginType represents the type of plugin
type PluginType string

const (
	// PluginTypePreProcessor processes requests before sending to AI
	PluginTypePreProcessor PluginType = "preprocessor"
	// PluginTypePostProcessor processes responses after receiving from AI
	PluginTypePostProcessor PluginType = "postprocessor"
	// PluginTypeAnalyzer provides code analysis capabilities
	PluginTypeAnalyzer PluginType = "analyzer"
	// PluginTypeFormatter formats code output
	PluginTypeFormatter PluginType = "formatter"
	// PluginTypeValidator validates code
	PluginTypeValidator PluginType = "validator"
	// PluginTypeTransformer transforms code structure
	PluginTypeTransformer PluginType = "transformer"
	// PluginTypeOptimizer optimizes code performance
	PluginTypeOptimizer PluginType = "optimizer"
	// PluginTypeSecurityChecker checks security vulnerabilities
	PluginTypeSecurityChecker PluginType = "security_checker"
	// PluginTypeDocumentationGenerator generates documentation
	PluginTypeDocumentationGenerator PluginType = "documentation_generator"
	// PluginTypeTestGenerator generates test code
	PluginTypeTestGenerator PluginType = "test_generator"
)

// PluginPriority defines plugin execution priority
type PluginPriority int

const (
	// PriorityLowest is the lowest priority
	PriorityLowest PluginPriority = 0
	// PriorityLow is low priority
	PriorityLow PluginPriority = 25
	// PriorityNormal is normal priority
	PriorityNormal PluginPriority = 50
	// PriorityHigh is high priority
	PriorityHigh PluginPriority = 75
	// PriorityHighest is the highest priority
	PriorityHighest PluginPriority = 100
)

// Plugin defines the interface for Codex plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Type returns the plugin type
	Type() PluginType

	// Priority returns the execution priority
	Priority() PluginPriority

	// IsEnabled returns whether the plugin is enabled
	IsEnabled() bool

	// Execute executes the plugin
	Execute(ctx context.Context, data interface{}) (interface{}, error)

	// Validate validates plugin configuration
	Validate() error
}

// PreProcessorPlugin processes requests before sending to AI
type PreProcessorPlugin interface {
	Plugin
	ProcessRequest(ctx context.Context, req *CodexRequest) (*CodexRequest, error)
}

// PostProcessorPlugin processes responses after receiving from AI
type PostProcessorPlugin interface {
	Plugin
	ProcessResponse(ctx context.Context, resp *CodexResponse) (*CodexResponse, error)
}

// AnalyzerPlugin provides code analysis capabilities
type AnalyzerPlugin interface {
	Plugin
	AnalyzeCode(ctx context.Context, code string, language CodexLanguage) (*CodeAnalysisResult, error)
}

// FormatterPlugin formats code output
type FormatterPlugin interface {
	Plugin
	FormatCode(ctx context.Context, code string, language CodexLanguage) (string, error)
}

// ValidatorPlugin validates code
type ValidatorPlugin interface {
	Plugin
	ValidateCode(ctx context.Context, code string, language CodexLanguage) (*ValidationResult, error)
}

// TransformerPlugin transforms code structure
type TransformerPlugin interface {
	Plugin
	TransformCode(ctx context.Context, code string, language CodexLanguage, transformation string) (string, error)
}

// OptimizerPlugin optimizes code performance
type OptimizerPlugin interface {
	Plugin
	OptimizeCode(ctx context.Context, code string, language CodexLanguage) (string, []string, error)
}

// SecurityCheckerPlugin checks security vulnerabilities
type SecurityCheckerPlugin interface {
	Plugin
	CheckSecurity(ctx context.Context, code string, language CodexLanguage) ([]SecurityIssue, error)
}

// DocumentationGeneratorPlugin generates documentation
type DocumentationGeneratorPlugin interface {
	Plugin
	GenerateDocumentation(ctx context.Context, code string, language CodexLanguage, format string) (string, error)
}

// TestGeneratorPlugin generates test code
type TestGeneratorPlugin interface {
	Plugin
	GenerateTests(ctx context.Context, code string, language CodexLanguage, testFramework string) (string, error)
}

// CodeAnalysisResult contains code analysis results
type CodeAnalysisResult struct {
	Complexity      int                      `json:"complexity"`
	SecurityIssues  []SecurityIssue          `json:"security_issues"`
	PerformanceIssues []PerformanceIssue     `json:"performance_issues"`
	CodeQuality     CodeQualityMetrics      `json:"code_quality"`
	Suggestions     []string                 `json:"suggestions"`
	Metadata        map[string]interface{}   `json:"metadata,omitempty"`
}

// SecurityIssue represents a security issue found in code
type SecurityIssue struct {
	Severity    string `json:"severity"` // "low", "medium", "high", "critical"
	Type        string `json:"type"`     // "sql_injection", "xss", "auth_bypass", etc.
	Description string `json:"description"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Fix         string `json:"fix,omitempty"`
}

// PerformanceIssue represents a performance issue found in code
type PerformanceIssue struct {
	Severity    string `json:"severity"`
	Type        string `json:"type"` // "n_plus_one", "inefficient_algorithm", "memory_leak", etc.
	Description string `json:"description"`
	Line        int    `json:"line,omitempty"`
	Impact      string `json:"impact,omitempty"`
	Fix         string `json:"fix,omitempty"`
}

// CodeQualityMetrics contains code quality metrics
type CodeQualityMetrics struct {
	MaintainabilityIndex float64 `json:"maintainability_index"` // 0-100
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	CodeDuplication      float64 `json:"code_duplication"` // percentage
	TestCoverage         float64 `json:"test_coverage,omitempty"` // percentage
	DocumentationCoverage float64 `json:"documentation_coverage,omitempty"` // percentage
}

// ValidationResult contains code validation results
type ValidationResult struct {
	Valid      bool     `json:"valid"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// PluginManager manages Codex plugins
type PluginManager struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
	}
}

// RegisterPlugin registers a plugin
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.plugins[plugin.Name()] = plugin
	return nil
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// GetPluginsByType returns all plugins of a specific type
func (pm *PluginManager) GetPluginsByType(pluginType PluginType) []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]Plugin, 0)
	for _, plugin := range pm.plugins {
		if plugin.Type() == pluginType && plugin.IsEnabled() {
			plugins = append(plugins, plugin)
		}
	}

	// Sort by priority (highest first)
	for i := 0; i < len(plugins)-1; i++ {
		for j := i + 1; j < len(plugins); j++ {
			if plugins[i].Priority() < plugins[j].Priority() {
				plugins[i], plugins[j] = plugins[j], plugins[i]
			}
		}
	}

	return plugins
}

// ExecutePreProcessors executes all pre-processor plugins
func (pm *PluginManager) ExecutePreProcessors(ctx context.Context, req *CodexRequest) (*CodexRequest, error) {
	plugins := pm.GetPluginsByType(PluginTypePreProcessor)

	result := req
	for _, plugin := range plugins {
		if preProcessor, ok := plugin.(PreProcessorPlugin); ok {
			processed, err := preProcessor.ProcessRequest(ctx, result)
			if err != nil {
				return nil, fmt.Errorf("preprocessor %s failed: %w", plugin.Name(), err)
			}
			result = processed
		}
	}

	return result, nil
}

// ExecutePostProcessors executes all post-processor plugins
func (pm *PluginManager) ExecutePostProcessors(ctx context.Context, resp *CodexResponse) (*CodexResponse, error) {
	plugins := pm.GetPluginsByType(PluginTypePostProcessor)

	result := resp
	for _, plugin := range plugins {
		if postProcessor, ok := plugin.(PostProcessorPlugin); ok {
			processed, err := postProcessor.ProcessResponse(ctx, result)
			if err != nil {
				return nil, fmt.Errorf("postprocessor %s failed: %w", plugin.Name(), err)
			}
			result = processed
		}
	}

	return result, nil
}

// ExecuteAnalyzers executes all analyzer plugins
func (pm *PluginManager) ExecuteAnalyzers(ctx context.Context, code string, language CodexLanguage) ([]*CodeAnalysisResult, error) {
	plugins := pm.GetPluginsByType(PluginTypeAnalyzer)

	results := make([]*CodeAnalysisResult, 0)
	for _, plugin := range plugins {
		if analyzer, ok := plugin.(AnalyzerPlugin); ok {
			result, err := analyzer.AnalyzeCode(ctx, code, language)
			if err != nil {
				return nil, fmt.Errorf("analyzer %s failed: %w", plugin.Name(), err)
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// ExecuteFormatters executes all formatter plugins
func (pm *PluginManager) ExecuteFormatters(ctx context.Context, code string, language CodexLanguage) (string, error) {
	plugins := pm.GetPluginsByType(PluginTypeFormatter)

	result := code
	for _, plugin := range plugins {
		if formatter, ok := plugin.(FormatterPlugin); ok {
			formatted, err := formatter.FormatCode(ctx, result, language)
			if err != nil {
				return "", fmt.Errorf("formatter %s failed: %w", plugin.Name(), err)
			}
			result = formatted
		}
	}

	return result, nil
}

// ExecuteValidators executes all validator plugins
func (pm *PluginManager) ExecuteValidators(ctx context.Context, code string, language CodexLanguage) ([]*ValidationResult, error) {
	plugins := pm.GetPluginsByType(PluginTypeValidator)

	results := make([]*ValidationResult, 0)
	for _, plugin := range plugins {
		if validator, ok := plugin.(ValidatorPlugin); ok {
			result, err := validator.ValidateCode(ctx, code, language)
			if err != nil {
				return nil, fmt.Errorf("validator %s failed: %w", plugin.Name(), err)
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// ExecuteSecurityCheckers executes all security checker plugins
func (pm *PluginManager) ExecuteSecurityCheckers(ctx context.Context, code string, language CodexLanguage) ([]SecurityIssue, error) {
	plugins := pm.GetPluginsByType(PluginTypeSecurityChecker)

	allIssues := make([]SecurityIssue, 0)
	for _, plugin := range plugins {
		if checker, ok := plugin.(SecurityCheckerPlugin); ok {
			issues, err := checker.CheckSecurity(ctx, code, language)
			if err != nil {
				return nil, fmt.Errorf("security checker %s failed: %w", plugin.Name(), err)
			}
			allIssues = append(allIssues, issues...)
		}
	}

	return allIssues, nil
}

// ExecuteOptimizers executes all optimizer plugins
func (pm *PluginManager) ExecuteOptimizers(ctx context.Context, code string, language CodexLanguage) (string, []string, error) {
	plugins := pm.GetPluginsByType(PluginTypeOptimizer)

	result := code
	suggestions := make([]string, 0)

	for _, plugin := range plugins {
		if optimizer, ok := plugin.(OptimizerPlugin); ok {
			optimized, opts, err := optimizer.OptimizeCode(ctx, result, language)
			if err != nil {
				return "", nil, fmt.Errorf("optimizer %s failed: %w", plugin.Name(), err)
			}
			result = optimized
			suggestions = append(suggestions, opts...)
		}
	}

	return result, suggestions, nil
}

// ExecuteDocumentationGenerators executes all documentation generator plugins
func (pm *PluginManager) ExecuteDocumentationGenerators(ctx context.Context, code string, language CodexLanguage, format string) (string, error) {
	plugins := pm.GetPluginsByType(PluginTypeDocumentationGenerator)

	if len(plugins) == 0 {
		return "", fmt.Errorf("no documentation generator plugins available")
	}

	// Use the first available generator
	plugin := plugins[0]
	if generator, ok := plugin.(DocumentationGeneratorPlugin); ok {
		return generator.GenerateDocumentation(ctx, code, language, format)
	}

	return "", fmt.Errorf("plugin %s does not implement DocumentationGeneratorPlugin", plugin.Name())
}

// ExecuteTestGenerators executes all test generator plugins
func (pm *PluginManager) ExecuteTestGenerators(ctx context.Context, code string, language CodexLanguage, testFramework string) (string, error) {
	plugins := pm.GetPluginsByType(PluginTypeTestGenerator)

	if len(plugins) == 0 {
		return "", fmt.Errorf("no test generator plugins available")
	}

	// Use the first available generator
	plugin := plugins[0]
	if generator, ok := plugin.(TestGeneratorPlugin); ok {
		return generator.GenerateTests(ctx, code, language, testFramework)
	}

	return "", fmt.Errorf("plugin %s does not implement TestGeneratorPlugin", plugin.Name())
}

// ExecuteTransformers executes all transformer plugins
func (pm *PluginManager) ExecuteTransformers(ctx context.Context, code string, language CodexLanguage, transformation string) (string, error) {
	plugins := pm.GetPluginsByType(PluginTypeTransformer)

	result := code
	for _, plugin := range plugins {
		if transformer, ok := plugin.(TransformerPlugin); ok {
			transformed, err := transformer.TransformCode(ctx, result, language, transformation)
			if err != nil {
				return "", fmt.Errorf("transformer %s failed: %w", plugin.Name(), err)
			}
			result = transformed
		}
	}

	return result, nil
}

// ListPlugins returns all registered plugins
func (pm *PluginManager) ListPlugins() []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}
