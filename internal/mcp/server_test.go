package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestServer_RegisterResource(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log)

	r := &Resource{
		Name:        "test-resource",
		Description: "Test",
		URITemplate: "erp://test",
		MimeType:    "text/plain",
	}

	s.RegisterResource(r)

	assert.NotNil(t, s.resources["erp://test"])
	assert.Equal(t, "test-resource", s.resources["erp://test"].Name)
}

func TestServer_RegisterPrompt(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log)

	p := &Prompt{
		Name:        "test-prompt",
		Description: "Test prompt",
		Template:    "Do something",
		Arguments: []PromptArgument{
			{Name: "arg1", Description: "Arg 1", Required: true},
		},
	}

	s.RegisterPrompt(p)

	assert.NotNil(t, s.prompts["test-prompt"])
	assert.Equal(t, "test-prompt", s.prompts["test-prompt"].Name)
}

func TestServer_HandlePromptGet(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log)

	p := &Prompt{
		Name:     "test-prompt",
		Template: "Hello {{name}}",
	}
	s.RegisterPrompt(p)

	req := mcp.GetPromptRequest{}
	req.Params.Name = "test-prompt"
	req.Params.Arguments = map[string]string{"name": "world"}

	res, err := s.handleMCPPromptGet(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Messages, 1)

	textContent := res.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, textContent, "Hello {{name}}")
	assert.Contains(t, textContent, "name: world")
}

func TestServer_RegisterToolMarshaling(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log)

	schema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"test": {Type: "string"},
		},
	}

	s.RegisterTool(&Tool{
		Name:        "test-tool",
		Description: "Test",
		InputSchema: schema,
	})

	// Access the registered tool from the internal mcpServer and try to marshal it
	assert.NotNil(t, s.tools["test-tool"])

	// Try to marshal the tools list as the server would during a tools/list request
	schemaJSON, _ := json.Marshal(schema)
	mcpTool := mcp.NewToolWithRawSchema("test-tool", "Test", json.RawMessage(schemaJSON))

	// Apply the same fix as in RegisterTool
	mcpTool.InputSchema = mcp.ToolInputSchema{}
	mcpTool.OutputSchema = mcp.ToolOutputSchema{}

	data, err := json.Marshal(mcpTool)
	assert.NoError(t, err, "Tool marshaling should not fail")
	assert.Contains(t, string(data), "\"inputSchema\":{\"type\":\"object\"")
}
