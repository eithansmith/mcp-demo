## MCP Demo
A simple MCP server that demonstrates how to use the [mcp-go](https://github.com/mark3labs/mcp-go) library.

## Setup Instructions
1.  Pull this repository (using git clone or download zip)
2.  Install [Go](https://golang.org/dl/) and [Claude Desktop](https://www.claude.com/download)
3.  Run `go run main.go` (or `go build main.go && ./main` to ensure the binary is built)
4.  Configure your server by creating a Claude Desktop config file:
    - Windows: %APPDATA%\Claude\claude_desktop_config.json
    - macOS: ~/Library/Application Support/Claude/claude_desktop_config.json
5.  Add the following to that JSON file. Replace the username_here to match your username.  Note that these instructions link to a batch file for Windows users. If you are on a different OS, please use the appropriate substitution.
    ```json
    {
        "mcpServers": {
           "demoServer": {
              "command": "C:/Users/username_here/mcp-demo/run-mcp.bat"
           }
        }
    } 
6.  Start Claude Desktop. Test your tool by asking Claude: "Use the hello_world tool to greet someone"

## How It Works

1. The server initializes with tool capabilities enabled
2. A `hello_world` tool is registered with a name parameter
3. When a client calls the tool, the `helloHandler` function processes the request
4. The handler validates the name parameter and returns a greeting message
5. Communication happens over stdio following the MCP protocol

## Development

To extend this demo:

1. Add new tools using `mcp.NewTool()`
2. Register tool handlers with `s.AddTool()`
3. Implement handler functions that return `*mcp.CallToolResult`

Please visit the [mcp-go quickstart](https://mcp-go.dev/quick-start) for further information on how to use the mcp-go library.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
