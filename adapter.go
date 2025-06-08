// Package langchaingo_mcp_adapter provides an adapter between LangChain Go and MCP servers.
package langchaingo_mcp_adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	langchaingoTools "github.com/tmc/langchaingo/tools"
)

// mcpTool implements the langchaingoTools.Tool interface for MCP tools.
type mcpTool struct {
	name        string
	description string
	inputSchema []byte
	client      client.MCPClient
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

	req := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	req.Params.Name = t.name

	var args map[string]interface{}
	err := json.Unmarshal([]byte(input), &args)
	if err != nil {
		return "call the tool error: input must be valid json, retry tool calling with correct json", nil
	}
	req.Params.Arguments = args

	res, err := t.client.CallTool(ctx, req)
	if err != nil {
		return fmt.Sprintf("call the tool error: %s", err), nil
	}

	return res.Content[0].(mcp.TextContent).Text, nil
}

// MCPAdapter adapts an MCP client to the LangChain Go tools interface.
type MCPAdapter struct {
	client  client.MCPClient
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
func New(client client.MCPClient, opts ...Option) (*MCPAdapter, error) {
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "langchaingo-mcp-adapter",
		Version: "1.0.1",
	}

	initResult, err := client.Initialize(context.Background(), initRequest)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	slog.Debug(
		"Initialized with server",
		"name",
		initResult.ServerInfo.Name,
		"version",
		initResult.ServerInfo.Version,
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
func newLangchaingoTool(name, description string, inputSchema map[string]any, client client.MCPClient, timeout time.Duration) (langchaingoTools.Tool, error) {
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
func NewToolForTesting(name, description string, inputSchema map[string]any, client client.MCPClient) (langchaingoTools.Tool, error) {
	return newLangchaingoTool(name, description, inputSchema, client, 30*time.Second)
}
