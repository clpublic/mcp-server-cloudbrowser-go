package main

import (
	"context"
	"log"
	"mcp-server-cloudbrowser-go/mcp/tools"
	"os"

	"github.com/dreamsxin/mcp-go/server"
)

// AuthFromEnv extracts the auth token from the environment
func AuthFromEnv(ctx context.Context) context.Context {
	sessionId := os.Getenv("SESSION_ID")
	apiKey := os.Getenv("API_KEY")
	log.Println("AuthFromEnv", sessionId, apiKey)
	ctx = tools.SetValueToContext(ctx, tools.SessionIDKey, sessionId)
	ctx = tools.SetValueToContext(ctx, tools.ApiKeyKey, apiKey)

	return ctx
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create MCP server
	s := server.NewMCPServer(
		"Cloud Browser",
		"1.0.0",
		server.WithLogging(),
	)

	// // Add tool handler
	// s.AddTool(tool, helloHandler)
	tools.InitTools(s)

	// Start the stdio server
	if err := server.ServeStdio(s, server.WithStdioContextFunc(AuthFromEnv)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
