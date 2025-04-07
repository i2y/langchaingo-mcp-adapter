# LangChainGo MCP Adapter

A Go adapter that bridges LangChain Go tools with Model Context Protocol (MCP) servers.

## Overview

This adapter allows you to use tools defined on an MCP server with the LangChain Go library. It implements the necessary interfaces to integrate MCP tools seamlessly with LangChain Go's agent infrastructure.

## Features

- Connect to any MCP server
- Automatically discover MCP tools from a specified MCP server and make them available to LangChain Go
- Wrap MCP tools as LangChain Go tools

## Installation

```bash
go get github.com/i2y/langchaingo-mcp-adapter
```

## Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/i2y/langchaingo-mcp-adapter"
    "github.com/mark3labs/mcp-go/client"
    "github.com/tmc/langchaingo/agents"
    "github.com/tmc/langchaingo/chains"
    "github.com/tmc/langchaingo/llms/googleai"
    "github.com/tmc/langchaingo/tools"
)

func main() {
    // Create an MCP client using stdio
    mcpClient, err := client.NewStdioMCPClient(
        "./a-mcp-sever",  // Path to an MCP server
        nil,              // Additional environment variables if needed
    )
    if err != nil {
        log.Fatalf("Failed to create MCP client: %v", err)
    }
    defer mcpClient.Close()

    // Create the adapter
    adapter, err := langchaingo_mcp_adapter.New(mcpClient)
    if err != nil {
        log.Fatalf("Failed to create adapter: %v", err)
    }

    // Get all tools from MCP server
    mcpTools, err := adapter.Tools()
    if err != nil {
        log.Fatalf("Failed to get tools: %v", err)
    }

    ctx := context.Backgorund()

    // Create a Google AI LLM client
    llm, err := googleai.New(
        ctx,
        googleai.WithDefaultModel("gemini-2.0-flash"),
        googleai.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
    )
    if err != nil {
        log.Fatalf("Create Google AI client: %v", err)
    }

    // Create a agent with the tools
    agent := agents.NewOneShotAgent(
        llm,
        mcpTools,
        agents.WithMaxIterations(3),
    )
    executor := agents.NewExecutor(agent)

    // Use the agent
    question := "Can you help me analyze this data using the available tools?"
    result, err := chains.Run(
        ctx,
        executor,
        question,
    )
    if err != nil {
        log.Fatalf("Agent execution error: %v", err)
    }

    log.Printf("Agent result: %s", result)
}
```

## Example Applications

See the `example` directory for a complete example:

- `example/agent`: Demonstrates how to use the adapter with an LLM agent
- `example/server`: A minimal MCP server example

The mcp-curl server in this sample is based on the code from [this blog](https://k33g.hashnode.dev/creating-an-mcp-server-in-go-and-serving-it-with-docker).

## Requirements

- Go 1.23 or higher
- [tmc/langchaingo](https://github.com/tmc/langchaingo)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
