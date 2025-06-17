package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Log to stderr to avoid interfering with stdio protocol
	log.SetOutput(os.Stderr)
	
	// Create MCP server
	s := server.NewMCPServer(
		"mcp-curl",
		"1.0.0",
	)

	// Add a tool
	tool := mcp.NewTool("fetch_url",
		mcp.WithDescription("fetch content from a URL"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL of the webpage to fetch"),
		),
	)

	// Add a tool handler
	s.AddTool(tool, fetchURLHandler)

	log.Println("🚀 Server starting...")
	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Printf("😡 Server error: %v\n", err)
		os.Exit(1)
	}
	log.Println("👋 Server stopped")
}

func fetchURLHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("arguments must be an object"), nil
	}
	
	url, ok := args["url"].(string)
	if !ok {
		return mcp.NewToolResultError("url must be a string"), nil
	}

	// Use Go's HTTP client instead of curl for cross-platform compatibility
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content := string(body)
	return mcp.NewToolResultText(content), nil
}
