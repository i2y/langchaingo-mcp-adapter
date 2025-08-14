// Package langchaingo_mcp_adapter provides an adapter between LangChain Go and MCP servers.
package langchaingo_mcp_adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	langchaingoTools "github.com/tmc/langchaingo/tools"
)

// MCPClientInterface defines the interface for MCP client operations used by this adapter.
// This interface is a subset of client.MCPClient that includes only the exported methods we need.
// The client.MCPClient from github.com/mark3labs/mcp-go/client package implements this interface.
type MCPClientInterface interface {
	Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// mcpTool implements the langchaingoTools.Tool interface for MCP tools.
type mcpTool struct {
	name        string
	description string
	inputSchema []byte
	client      MCPClientInterface
	timeout     time.Duration
}

// Name returns the name of the tool.
func (t *mcpTool) Name() string {
	return t.name
}

// Description returns the description of the tool along with its input schema.
func (t *mcpTool) Description() string {
	return t.description + "\n The input schema is: " + string(t.inputSchema)
}

// Call invokes the MCP tool with the given input and returns the result.
func (t *mcpTool) Call(ctx context.Context, input string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Debug logging to help identify JSON parsing issues
	slog.Debug("mcpTool.Call received input",
		"tool", t.name,
		"input_length", len(input),
		"input", input)

	req := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	req.Params.Name = t.name

	var args map[string]interface{}
	err := json.Unmarshal([]byte(input), &args)
	if err != nil {
		// Log the actual error and input for debugging
		slog.Error("Failed to unmarshal input JSON",
			"tool", t.name,
			"error", err,
			"input", input,
			"input_bytes", []byte(input))
		return fmt.Sprintf("call the tool error: input must be valid json (error: %v), retry tool calling with correct json", err), nil
	}
	req.Params.Arguments = args

	res, err := t.client.CallTool(ctx, req)
	if err != nil {
		slog.Error("MCP tool call failed",
			"tool", t.name,
			"error", err)
		return fmt.Sprintf("call the tool error: %s", err), nil
	}

	// Add safety check for response content
	if len(res.Content) == 0 {
		return "", fmt.Errorf("MCP tool returned empty content")
	}

	textContent, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("MCP tool returned unexpected content type: %T", res.Content[0])
	}

	return textContent.Text, nil
}

// MCPAdapter adapts an MCP client to the LangChain Go tools interface.
type MCPAdapter struct {
	client  MCPClientInterface
	timeout time.Duration
}

// Option defines a functional option type for configuring MCPAdapter.
type Option func(*MCPAdapter)

// WithToolTimeout sets the timeout for tool calls.
func WithToolTimeout(timeout time.Duration) Option {
	return func(a *MCPAdapter) {
		a.timeout = timeout
	}
}

// New creates a new MCPAdapter instance with the given MCP client.
// It initializes the connection with the MCP server.
// Optional parameters can be passed using functional options.
func New(client MCPClientInterface, opts ...Option) (*MCPAdapter, error) {
	if client == nil {
		return nil, fmt.Errorf("MCP client cannot be nil")
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "langchaingo-mcp-adapter",
		Version: "1.0.1",
	}

	slog.Debug("Initializing MCP adapter",
		"protocol_version", mcp.LATEST_PROTOCOL_VERSION,
		"client_name", "langchaingo-mcp-adapter")

	initResult, err := client.Initialize(context.Background(), initRequest)
	if err != nil {
		// Provide more context about the initialization failure
		slog.Error("Failed to initialize MCP client",
			"error", err,
			"protocol_version", mcp.LATEST_PROTOCOL_VERSION)
		return nil, fmt.Errorf("failed to initialize MCP client (this may be due to stdout contamination or server startup issues): %w", err)
	}

	if initResult == nil || initResult.ServerInfo.Name == "" {
		return nil, fmt.Errorf("invalid initialization result: server info is missing")
	}

	slog.Info(
		"Successfully initialized MCP adapter with server",
		"server_name", initResult.ServerInfo.Name,
		"server_version", initResult.ServerInfo.Version,
	)

	adapter := &MCPAdapter{
		client:  client,
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter, nil
}

// Tools returns a list of all available tools from the MCP server.
// Each tool is wrapped as a langchaingoTools.Tool.
func (a *MCPAdapter) Tools() ([]langchaingoTools.Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	toolsRequest := mcp.ListToolsRequest{}
	tools, err := a.client.ListTools(ctx, toolsRequest)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	var mcpTools []langchaingoTools.Tool

	for _, tool := range tools.Tools {
		slog.Debug("tool", "name", tool.Name, "description", tool.Description)

		mcpTool, err := newLangchaingoTool(tool.Name, tool.Description, tool.InputSchema.Properties, a.client, a.timeout)
		if err != nil {
			return nil, fmt.Errorf("new langchaingo tool: %w", err)
		}
		mcpTools = append(mcpTools, mcpTool)
	}

	return mcpTools, nil
}

// newLangchaingoTool creates a new langchaingo tool from MCP tool information.
func newLangchaingoTool(name, description string, inputSchema map[string]any, client MCPClientInterface, timeout time.Duration) (langchaingoTools.Tool, error) {
	jsonSchema, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal input schema: %w", err)
	}

	return &mcpTool{
		name:        name,
		description: description,
		inputSchema: jsonSchema,
		client:      client,
		timeout:     timeout,
	}, nil
}

// NewToolForTesting creates an mcpTool instance for testing purposes.
// This function is for testing only and should not be used in production applications.
func NewToolForTesting(name, description string, inputSchema map[string]any, client MCPClientInterface) (langchaingoTools.Tool, error) {
	return newLangchaingoTool(name, description, inputSchema, client, 30*time.Second)
}
