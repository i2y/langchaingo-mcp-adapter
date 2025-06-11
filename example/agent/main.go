package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
	langchaingoTools "github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/wikipedia"

	mcpadapter "github.com/i2y/langchaingo-mcp-adapter"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Create LLM based on available API keys
	llm, err := createLLM()
	if err != nil {
		return fmt.Errorf("create LLM: %w", err)
	}

	wp := wikipedia.New("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Start the MCP server
	mcpClient, err := startMCPServer()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer mcpClient.Close()

	fmt.Println("Creating MCP adapter...")
	mcpAdapter, err := mcpadapter.New(mcpClient, mcpadapter.WithToolTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("new mcp adapter: %w", err)
	}

	agentTools := []langchaingoTools.Tool{
		langchaingoTools.Calculator{},
		wp,
	}
	mcpTools, err := mcpAdapter.Tools()
	if err != nil {
		return fmt.Errorf("append tools: %w", err)
	}
	agentTools = append(agentTools, mcpTools...)

	agent := agents.NewOneShotAgent(llm,
		agentTools,
		agents.WithMaxIterations(3))
	executor := agents.NewExecutor(agent)

	question := "Could you provide a summary of https://raw.githubusercontent.com/docker-sa/01-build-image/refs/heads/main/main.go"
	fmt.Println("🔍 Question:", question)

	fmt.Println("🚀 Starting agent...")
	answer, err := chains.Run(context.Background(), executor, question)
	if err != nil {
		return fmt.Errorf("error running chains: %w", err)
	}

	fmt.Println("🎉 Answer:", answer)

	return nil
}


// createLLM creates an LLM instance based on available API keys
func createLLM() (llms.Model, error) {
	ctx := context.Background()

	// Check for Google AI
	if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		fmt.Println("🤖 Using Google AI (Gemini)")
		return googleai.New(ctx,
			googleai.WithDefaultModel("gemini-2.0-flash"),
			googleai.WithAPIKey(apiKey),
		)
	}

	// Check for OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		fmt.Println("🤖 Using OpenAI")
		return openai.New(
			openai.WithToken(apiKey),
			openai.WithModel("gpt-4"),
		)
	}

	// Check for Anthropic
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		fmt.Println("🤖 Using Anthropic (Claude)")
		return anthropic.New(
			anthropic.WithToken(apiKey),
			anthropic.WithModel("claude-3-5-sonnet-20241022"),
		)
	}

	return nil, fmt.Errorf("no LLM API key found. Please set one of: GOOGLE_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY")
}

// getExecutableName returns the executable name with proper extension for the OS
func getExecutableName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// startMCPServer starts the MCP server, building it if necessary.
// We build and use a binary instead of "go run" because go run can interfere
// with stdio communication, causing the MCP protocol initialization to hang.
func startMCPServer() (client.MCPClient, error) {
	serverDir := "../server"
	executableName := getExecutableName("mcp-server")
	serverBinary := filepath.Join(serverDir, executableName)
	serverSource := filepath.Join(serverDir, "main.go")
	
	// Check if binary exists
	if _, err := os.Stat(serverBinary); os.IsNotExist(err) {
		fmt.Println("Building MCP server binary...")
		cmd := exec.Command("go", "build", "-o", executableName, "main.go")
		cmd.Dir = serverDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("build server: %w\nOutput: %s", err, output)
		}
		fmt.Println("MCP server binary built successfully")
	}
	
	// Use the binary instead of go run to avoid stdio issues
	fmt.Printf("Starting MCP server from: %s\n", serverBinary)
	mcpClient, err := client.NewStdioMCPClient(serverBinary, []string{})
	if err != nil {
		// If binary fails, try go run as fallback with a warning
		fmt.Println("Warning: Failed to start binary, trying 'go run' (may hang due to stdio issues)")
		mcpClient, err = client.NewStdioMCPClient("go", []string{"run", serverSource})
		if err != nil {
			return nil, fmt.Errorf("start server: %w", err)
		}
	}
	
	return mcpClient, nil
}
