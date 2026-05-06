package mcp

import (
	"context"
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
