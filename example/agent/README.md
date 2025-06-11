# LangChain Go MCP Adapter Example

This example demonstrates how to use the LangChain Go MCP adapter with various LLM providers.

## Prerequisites

Set one of the supported LLM provider API keys:
- **Google AI (Gemini)**: Set `GOOGLE_API_KEY`
- **OpenAI**: Set `OPENAI_API_KEY`
- **Anthropic (Claude)**: Set `ANTHROPIC_API_KEY`

## Running the Example

1. Set your preferred LLM API key:
   ```bash
   # For Google AI
   export GOOGLE_API_KEY="your-api-key"
   
   # For OpenAI
   export OPENAI_API_KEY="your-api-key"
   
   # For Anthropic
   export ANTHROPIC_API_KEY="your-api-key"
   ```

2. Run the example from this directory:
   ```bash
   go run main.go
   ```

The example will automatically build and start the MCP server from `../server/main.go`.

## How it Works

1. The example automatically builds the MCP server (from `../server/main.go`) as a binary on first run
   - On Windows, the binary will be named `mcp-server.exe`
   - On macOS/Linux, it will be named `mcp-server`
2. The MCP server is started as a subprocess using stdio communication
3. The MCP adapter connects to this server and discovers available tools (in this case, a `fetch_url` tool)
4. The discovered tools are combined with built-in LangChain tools (Calculator, Wikipedia)
5. An agent is created with these tools and processes the user's question
6. The agent can use the fetch_url tool to retrieve web content and provide summaries

## Cross-Platform Compatibility

This example is designed to work on Windows, macOS, and Linux:
- The MCP server uses Go's `net/http` package instead of external tools like `curl`
- Binary names are automatically adjusted for the target OS
- All file paths use Go's `filepath` package for proper path handling

## Troubleshooting

If you get an LLM error:
1. Make sure you have set one of the required API keys
2. Check that your API key is valid

If the MCP server fails to start:
1. Make sure you have Go installed and in your PATH
2. Check that the `../server/main.go` file exists
