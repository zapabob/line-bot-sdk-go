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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// MCPServerType represents the type of MCP server
type MCPServerType string

const (
	// MCPServerGeminiCLI represents Gemini CLI MCP server
	MCPServerGeminiCLI MCPServerType = "gemini-cli"
	// MCPServerClaudeCode represents Claude Code MCP server
	MCPServerClaudeCode MCPServerType = "claude-code"
)

// MCPServerConfig contains configuration for an MCP server
type MCPServerConfig struct {
	Type        MCPServerType `json:"type"`
	Command     string        `json:"command,omitempty"`     // Command to start MCP server
	Args        []string      `json:"args,omitempty"`        // Arguments for the command
	URL         string        `json:"url,omitempty"`         // URL for SSE-based MCP server
	APIKey      string        `json:"api_key,omitempty"`     // API key if required
	Enabled     bool          `json:"enabled"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	MaxRetries  int           `json:"max_retries,omitempty"`
}

// MCPTool represents a tool available through MCP
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// MCPResource represents a resource available through MCP
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// MCPClient handles communication with MCP servers
type MCPClient interface {
	// Connect connects to the MCP server
	Connect(ctx context.Context) error

	// Disconnect disconnects from the MCP server
	Disconnect() error

	// ListTools lists available tools
	ListTools(ctx context.Context) ([]MCPTool, error)

	// CallTool calls a tool with given arguments
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*MCPToolResult, error)

	// ListResources lists available resources
	ListResources(ctx context.Context) ([]MCPResource, error)

	// ReadResource reads a resource
	ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error)

	// IsConnected returns whether the client is connected
	IsConnected() bool

	// Type returns the MCP server type
	Type() MCPServerType
}

// MCPToolResult represents the result of a tool call
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent represents content in an MCP response
type MCPContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// MCPResourceContent represents the content of a resource
type MCPResourceContent struct {
	URI      string      `json:"uri"`
	MimeType string      `json:"mimeType"`
	Text     string      `json:"text,omitempty"`
	Blob     string      `json:"blob,omitempty"`
}

// StdioMCPClient implements MCP client for stdio-based servers
type StdioMCPClient struct {
	config     MCPServerConfig
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	connected   bool
	mu          sync.RWMutex
	requestID   int64
	requestMu   sync.Mutex
	pendingReqs map[int64]chan *MCPResponse
}

// MCPResponse represents a response from MCP server
type MCPResponse struct {
	ID      int64                  `json:"id,omitempty"`
	Result  interface{}            `json:"result,omitempty"`
	Error   *MCPError              `json:"error,omitempty"`
	Method  string                 `json:"method,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPError represents an error from MCP server
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewStdioMCPClient creates a new stdio-based MCP client
func NewStdioMCPClient(config MCPServerConfig) *StdioMCPClient {
	return &StdioMCPClient{
		config:      config,
		pendingReqs: make(map[int64]chan *MCPResponse),
	}
}

// Connect connects to the MCP server via stdio
func (c *StdioMCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Start the MCP server process
	cmd := exec.CommandContext(ctx, c.config.Command, c.config.Args...)
	
	var err error
	c.stdin, err = cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stderr, err = cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	c.cmd = cmd
	c.connected = true

	// Start response reader
	go c.readResponses()

	// Initialize the connection
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "line-bot-codex",
				"version": "1.0.0",
			},
		},
	}

	if err := c.sendRequest(initReq); err != nil {
		c.Disconnect()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// Disconnect disconnects from the MCP server
func (c *StdioMCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	if c.cmd != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	return nil
}

// ListTools lists available tools
func (c *StdioMCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      c.getNextRequestID(),
		"method":  "tools/list",
	}

	resp, err := c.sendRequestAndWait(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Parse response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	toolsData, ok := result["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tools format")
	}

	tools := make([]MCPTool, 0, len(toolsData))
	for _, toolData := range toolsData {
		toolMap, ok := toolData.(map[string]interface{})
		if !ok {
			continue
		}

		tool := MCPTool{
			Name:        getString(toolMap, "name"),
			Description: getString(toolMap, "description"),
		}

		if inputSchema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
			tool.InputSchema = inputSchema
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// CallTool calls a tool with given arguments
func (c *StdioMCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*MCPToolResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      c.getNextRequestID(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	resp, err := c.sendRequestAndWait(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Parse response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	contentData, ok := result["content"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid content format")
	}

	content := make([]MCPContent, 0, len(contentData))
	for _, item := range contentData {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		content = append(content, MCPContent{
			Type: getString(itemMap, "type"),
			Text: getString(itemMap, "text"),
			Data: getString(itemMap, "data"),
			URI:  getString(itemMap, "uri"),
		})
	}

	return &MCPToolResult{
		Content: content,
		IsError: getBool(result, "isError"),
	}, nil
}

// ListResources lists available resources
func (c *StdioMCPClient) ListResources(ctx context.Context) ([]MCPResource, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      c.getNextRequestID(),
		"method":  "resources/list",
	}

	resp, err := c.sendRequestAndWait(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Parse response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	resourcesData, ok := result["resources"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid resources format")
	}

	resources := make([]MCPResource, 0, len(resourcesData))
	for _, resData := range resourcesData {
		resMap, ok := resData.(map[string]interface{})
		if !ok {
			continue
		}

		resources = append(resources, MCPResource{
			URI:         getString(resMap, "uri"),
			Name:        getString(resMap, "name"),
			Description: getString(resMap, "description"),
			MimeType:    getString(resMap, "mimeType"),
		})
	}

	return resources, nil
}

// ReadResource reads a resource
func (c *StdioMCPClient) ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      c.getNextRequestID(),
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": uri,
		},
	}

	resp, err := c.sendRequestAndWait(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Parse response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return &MCPResourceContent{
		URI:      getString(result, "uri"),
		MimeType: getString(result, "mimeType"),
		Text:     getString(result, "text"),
		Blob:     getString(result, "blob"),
	}, nil
}

// IsConnected returns whether the client is connected
func (c *StdioMCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Type returns the MCP server type
func (c *StdioMCPClient) Type() MCPServerType {
	return c.config.Type
}

// Helper methods
func (c *StdioMCPClient) getNextRequestID() int64 {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requestID++
	return c.requestID
}

func (c *StdioMCPClient) sendRequest(req map[string]interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.stdin == nil {
		return fmt.Errorf("not connected")
	}

	_, err = fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(data), string(data))
	return err
}

func (c *StdioMCPClient) sendRequestAndWait(req map[string]interface{}) (*MCPResponse, error) {
	reqID, ok := req["id"].(int64)
	if !ok {
		if idFloat, ok := req["id"].(float64); ok {
			reqID = int64(idFloat)
		} else {
			return nil, fmt.Errorf("invalid request ID")
		}
	}

	ch := make(chan *MCPResponse, 1)
	c.requestMu.Lock()
	c.pendingReqs[reqID] = ch
	c.requestMu.Unlock()

	defer func() {
		c.requestMu.Lock()
		delete(c.pendingReqs, reqID)
		c.requestMu.Unlock()
	}()

	if err := c.sendRequest(req); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(c.config.Timeout):
		return nil, fmt.Errorf("request timeout")
	}
}

func (c *StdioMCPClient) readResponses() {
	decoder := json.NewDecoder(c.stdout)
	for {
		var resp MCPResponse
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if resp.ID != 0 {
			c.requestMu.Lock()
			if ch, ok := c.pendingReqs[resp.ID]; ok {
				ch <- &resp
			}
			c.requestMu.Unlock()
		}
	}
}

// SSE MCP Client (for HTTP-based MCP servers)
type SSEMCPClient struct {
	config    MCPServerConfig
	client    *http.Client
	connected bool
	mu        sync.RWMutex
}

// NewSSEMCPClient creates a new SSE-based MCP client
func NewSSEMCPClient(config MCPServerConfig) *SSEMCPClient {
	return &SSEMCPClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Connect connects to the SSE-based MCP server
func (c *SSEMCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Test connection
	resp, err := c.client.Get(c.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	resp.Body.Close()

	c.connected = true
	return nil
}

// Disconnect disconnects from the MCP server
func (c *SSEMCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return nil
}

// ListTools lists available tools (SSE implementation)
func (c *SSEMCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	// SSE implementation would use HTTP POST requests
	// This is a simplified version
	return nil, fmt.Errorf("SSE MCP client not fully implemented")
}

// CallTool calls a tool (SSE implementation)
func (c *SSEMCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*MCPToolResult, error) {
	return nil, fmt.Errorf("SSE MCP client not fully implemented")
}

// ListResources lists available resources (SSE implementation)
func (c *SSEMCPClient) ListResources(ctx context.Context) ([]MCPResource, error) {
	return nil, fmt.Errorf("SSE MCP client not fully implemented")
}

// ReadResource reads a resource (SSE implementation)
func (c *SSEMCPClient) ReadResource(ctx context.Context, uri string) (*MCPResourceContent, error) {
	return nil, fmt.Errorf("SSE MCP client not fully implemented")
}

// IsConnected returns whether the client is connected
func (c *SSEMCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Type returns the MCP server type
func (c *SSEMCPClient) Type() MCPServerType {
	return c.config.Type
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

// MCPProviderAdapter adapts MCP clients to AIProvider interface
type MCPProviderAdapter struct {
	mcpClient MCPClient
	config    AIProviderConfig
	name      string
}

// NewMCPProviderAdapter creates a new MCP provider adapter
func NewMCPProviderAdapter(mcpClient MCPClient, config AIProviderConfig) *MCPProviderAdapter {
	return &MCPProviderAdapter{
		mcpClient: mcpClient,
		config:    config,
		name:      string(mcpClient.Type()),
	}
}

// Type returns the provider type
func (a *MCPProviderAdapter) Type() AIProviderType {
	switch a.mcpClient.Type() {
	case MCPServerGeminiCLI:
		return ProviderGoogle
	case MCPServerClaudeCode:
		return ProviderAnthropic
	default:
		return AIProviderType(a.name)
	}
}

// IsEnabled returns whether the provider is enabled
func (a *MCPProviderAdapter) IsEnabled() bool {
	return a.config.Enabled && a.mcpClient.IsConnected()
}

// ProcessRequest processes a codex request via MCP
func (a *MCPProviderAdapter) ProcessRequest(ctx context.Context, req CodexRequest) (*AIProviderResponse, error) {
	if !a.mcpClient.IsConnected() {
		if err := a.mcpClient.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
		}
	}

	// Build prompt for MCP
	prompt := buildPromptForMCP(req)

	// Find appropriate tool
	tools, err := a.mcpClient.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Use code generation tool if available
	var toolName string
	for _, tool := range tools {
		if strings.Contains(strings.ToLower(tool.Name), "code") ||
			strings.Contains(strings.ToLower(tool.Name), "generate") {
			toolName = tool.Name
			break
		}
	}

	if toolName == "" && len(tools) > 0 {
		toolName = tools[0].Name
	}

	if toolName == "" {
		return nil, fmt.Errorf("no tools available")
	}

	// Call tool
	arguments := map[string]interface{}{
		"prompt":   prompt,
		"language": string(req.Language),
		"mode":     string(req.Mode),
	}

	if req.Code != "" {
		arguments["code"] = req.Code
	}

	result, err := a.mcpClient.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	// Extract text content
	content := ""
	for _, item := range result.Content {
		if item.Type == "text" {
			content += item.Text + "\n"
		}
	}

	return &AIProviderResponse{
		Content:    strings.TrimSpace(content),
		Model:      a.config.Model,
		Provider:   a.Type(),
		TokensUsed: 0, // MCP doesn't provide token usage
		Metadata: map[string]interface{}{
			"mcp_server": a.name,
			"tool_used":  toolName,
		},
	}, nil
}

// ValidateConfig validates the provider configuration
func (a *MCPProviderAdapter) ValidateConfig() error {
	if a.mcpClient == nil {
		return fmt.Errorf("MCP client is required")
	}
	return nil
}

// GetCapabilities returns the capabilities of this provider
func (a *MCPProviderAdapter) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsCodeGeneration: true,
		SupportsCodeReview:      true,
		SupportsCodeFix:         true,
		SupportsCodeExplain:     true,
		SupportsCodeRefactor:    true,
		SupportedLanguages:      []string{"go", "python", "javascript", "typescript", "java", "rust"},
		MaxContextLength:        100000,
		SupportsStreaming:        false,
	}
}

// buildPromptForMCP builds a prompt for MCP tools
func buildPromptForMCP(req CodexRequest) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Language: %s\n", req.Language))
	prompt.WriteString(fmt.Sprintf("Mode: %s\n", req.Mode))

	switch req.Mode {
	case CodexModeGenerate:
		prompt.WriteString(fmt.Sprintf("Generate code for: %s\n", req.Prompt))
		if req.Context != "" {
			prompt.WriteString(fmt.Sprintf("Context: %s\n", req.Context))
		}
	case CodexModeReview:
		prompt.WriteString("Review the following code:\n")
		prompt.WriteString(req.Code)
	case CodexModeFix:
		prompt.WriteString("Fix bugs in the following code:\n")
		prompt.WriteString(req.Code)
	case CodexModeExplain:
		prompt.WriteString("Explain the following code:\n")
		prompt.WriteString(req.Code)
	case CodexModeRefactor:
		prompt.WriteString("Refactor the following code:\n")
		prompt.WriteString(req.Code)
	}

	return prompt.String()
}

