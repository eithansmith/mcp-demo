package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// PromptHandler is the function signature for prompt handlers
type PromptHandler func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error)

// PromptRegistry stores all registered prompts and their handlers
type PromptRegistry struct {
	prompts  map[string]mcp.Prompt
	handlers map[string]PromptHandler
}

func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts:  make(map[string]mcp.Prompt),
		handlers: make(map[string]PromptHandler),
	}
}

func (pr *PromptRegistry) Register(prompt mcp.Prompt, handler PromptHandler) {
	name := prompt.Name
	pr.prompts[name] = prompt
	pr.handlers[name] = handler
}

func (pr *PromptRegistry) Get(name string) (mcp.Prompt, PromptHandler, bool) {
	prompt, promptOk := pr.prompts[name]
	handler, handlerOk := pr.handlers[name]
	return prompt, handler, promptOk && handlerOk
}

func (pr *PromptRegistry) List() []mcp.Prompt {
	result := make([]mcp.Prompt, 0, len(pr.prompts))
	for _, prompt := range pr.prompts {
		result = append(result, prompt)
	}
	return result
}

// Global registry
var promptRegistry = NewPromptRegistry()

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Hello World Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
	)

	// Define a simple tool
	tool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)

	toolMath := mcp.NewTool("calculate",
		mcp.WithDescription("Perform arithmetic operations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Enum("add", "subtract", "multiply", "divide"),
			mcp.Description("The arithmetic operation to perform"),
		),
		mcp.WithNumber("x", mcp.Required(), mcp.Description("First number")),
		mcp.WithNumber("y", mcp.Required(), mcp.Description("Second number")),
	)

	// NEW: Tool to access prompts
	executePromptTool := mcp.NewTool("execute_prompt",
		mcp.WithDescription("Execute a prompt template with given arguments"),
		mcp.WithString("prompt_name",
			mcp.Required(),
			mcp.Description("Name of the prompt to execute (e.g., 'code_review')"),
		),
		mcp.WithString("code",
			mcp.Description("Code to review (required for code_review prompt)"),
		),
		mcp.WithString("language",
			mcp.Description("Programming language"),
		),
		mcp.WithString("focus",
			mcp.Description("Focus area: security, performance, readability, best-practices, or all"),
		),
	)

	// Tool to list available prompts
	listPromptsTool := mcp.NewTool("list_prompts",
		mcp.WithDescription("List all available prompt templates with their descriptions and required arguments"),
	)

	codeReviewPrompt := mcp.NewPrompt("code_review",
		mcp.WithPromptDescription("Review code for best practices, bugs, and improvements"),
		mcp.WithArgument("code",
			mcp.ArgumentDescription("The code to review"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("language",
			mcp.ArgumentDescription("Programming language (auto-detected if not specified)"),
		),
		mcp.WithArgument("focus",
			mcp.ArgumentDescription("Specific areas to focus on (e.g., security, performance, readability, best-practices, all)"),
		),
	)

	// Register prompts with their handlers
	promptRegistry.Register(codeReviewPrompt, handleCodeReview)

	// Add all prompts to server
	for name, prompt := range promptRegistry.prompts {
		handler := promptRegistry.handlers[name]
		localHandler := handler // Capture in closure

		s.AddPrompt(prompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return localHandler(ctx, req)
		})
	}

	// Add tool handler
	s.AddTool(tool, helloHandler)
	s.AddTool(toolMath, mathHandler)
	s.AddTool(executePromptTool, executePromptHandler)
	s.AddTool(listPromptsTool, listPromptsHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Generic prompt execution handler
func executePromptHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	promptName, ok := arguments["prompt_name"].(string)
	if !ok || promptName == "" {
		return mcp.NewToolResultError("prompt_name parameter is required"), nil
	}

	// Get prompt and handler from registry
	prompt, handler, exists := promptRegistry.Get(promptName)
	if !exists {
		availablePrompts := ""
		for _, p := range promptRegistry.List() {
			availablePrompts += fmt.Sprintf("\n- %s: %s", p.Name, p.Description)
		}
		return mcp.NewToolResultError(fmt.Sprintf("Unknown prompt '%s'. Available prompts:%s", promptName, availablePrompts)), nil
	}

	// Extract arguments object
	promptArgs := make(map[string]string)

	// First, try to get from nested "arguments" object
	if args, ok := arguments["arguments"].(map[string]interface{}); ok {
		for key, val := range args {
			if strVal, ok := val.(string); ok {
				promptArgs[key] = strVal
			}
		}
	}

	// Then, also check top-level parameters (for backwards compatibility)
	// This allows both calling styles to work
	for key, val := range arguments {
		if key == "prompt_name" {
			continue // Skip the prompt_name itself
		}
		if strVal, ok := val.(string); ok && strVal != "" {
			promptArgs[key] = strVal
		}
	}

	// Validate required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required {
			if _, exists := promptArgs[arg.Name]; !exists {
				return mcp.NewToolResultError(fmt.Sprintf("Required argument '%s' missing for prompt '%s'", arg.Name, promptName)), nil
			}
		}
	}

	// Build prompt request
	promptReq := mcp.GetPromptRequest{
		Params: struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments,omitempty"`
		}{
			Name:      promptName,
			Arguments: promptArgs,
		},
	}

	// Execute the prompt handler
	result, err := handler(context.Background(), promptReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Prompt execution failed: %v", err)), nil
	}

	// Format the prompt result as tool output
	if len(result.Messages) > 0 {
		message := result.Messages[0]

		// Type assert to get the actual text content
		var text string
		switch c := message.Content.(type) {
		case mcp.TextContent:
			text = c.Text
		case *mcp.TextContent:
			text = c.Text
		default:
			// Fallback if type assertion fails
			text = fmt.Sprintf("%v", message.Content)
		}

		mcp.NewToolResultError(fmt.Sprintf("=== Prompt: %s ===\n\n%s", result.Description, text))
	}

	return mcp.NewToolResultError("Prompt returned no content"), nil
}

// Handler to list all available prompts
func listPromptsHandler(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prompts := promptRegistry.List()

	if len(prompts) == 0 {
		return mcp.NewToolResultText("No prompts registered"), nil
	}

	output := "Available Prompts:\n\n"
	for _, prompt := range prompts {
		output += fmt.Sprintf("\U0001F4CB %s\n", prompt.Name)
		output += fmt.Sprintf("   Description: %s\n", prompt.Description)

		if len(prompt.Arguments) > 0 {
			output += "   Arguments:\n"
			for _, arg := range prompt.Arguments {
				required := ""
				if arg.Required {
					required = " (required)"
				}
				output += fmt.Sprintf("   - %s%s: %s\n", arg.Name, required, arg.Description)
			}
		}
		output += "\n"
	}

	return mcp.NewToolResultText(output), nil
}

func helloHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	name, ok := arguments["name"].(string)
	if !ok {
		return mcp.NewToolResultError("Error: name parameter is required and must be a string"), nil
	}

	return mcp.NewToolResultError(fmt.Sprintf("Hello, %s! \U0001F44B", name)), nil
}

func mathHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	arguments := request.GetArguments()
	operation, ok := arguments["operation"].(string)
	if !ok {
		return mcp.NewToolResultError("Error: operation parameter is required and must be a string"), nil
	}
	x, ok := arguments["x"].(float64)
	if !ok {
		return mcp.NewToolResultError("Error: x parameter is required and must be a number"), nil
	}
	y, ok := arguments["y"].(float64)
	if !ok {
		return mcp.NewToolResultError("Error: y parameter is required and must be a number"), nil
	}

	result := 0.0
	switch operation {
	case "add":
		result = x + y
	case "subtract":
		result = x - y
	case "multiply":
		result = x * y
	case "divide":
		{
			if y == 0 {
				return mcp.NewToolResultError("division by zero"), nil
			}
			result = x / y
		}
	default:
		return mcp.NewToolResultError(fmt.Sprintf("invalid operation: %s", operation)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Answer: %.2f", result)), nil
}

func handleCodeReview(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	code, ok := req.Params.Arguments["code"]
	if !ok || code == "" {
		return nil, fmt.Errorf("code parameter is required")
	}
	language := req.Params.Arguments["language"]
	if language == "" {
		language = "unknown"
	}
	focus := req.Params.Arguments["focus"]
	if focus == "" {
		focus = "all"
	}

	// Build the prompt based on focus area
	var instructions string
	switch focus {
	case "security":
		instructions = "Focus specifically on security vulnerabilities, input validation, and potential attack vectors."
	case "performance":
		instructions = "Focus on performance optimizations, algorithmic efficiency, and resource usage."
	case "readability":
		instructions = "Focus on code clarity, naming conventions, and maintainability."
	case "best-practices":
		instructions = "Focus on language-specific best practices and design patterns."
	default:
		instructions = "Provide a comprehensive review covering security, performance, readability, and best practices."
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Code review for %s code", language),
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.NewTextContent(
					fmt.Sprintf("Please review the following %s code:\n\n%s\n\nInstructions: %s\n\n...", language, code, instructions)),
			},
		},
	}, nil
}
