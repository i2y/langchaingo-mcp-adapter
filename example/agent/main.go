package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/googleai"
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
	llm, err := googleai.New(
		context.Background(),
		googleai.WithDefaultModel("gemini-2.0-flash"), // gemini-2.5-pro-exp-03-25"),
		googleai.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
	)
	if err != nil {
		return err
	}
	wp := wikipedia.New("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	mcpClient, err := client.NewStdioMCPClient("./mcp-curl", nil)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer mcpClient.Close()

	mcpAdapter, err := mcpadapter.New(mcpClient)
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
