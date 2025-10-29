## MCP Demo
A simple MCP server that demonstrates how to use the [mcp-go](https://github.com/mark3labs/mcp-go) library.
Please visit the [mcp-go quickstart](https://mcp-go.dev/quick-start) for more information.

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

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
