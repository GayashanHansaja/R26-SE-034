package idp

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nimendra/ERPBridge/internal/mcp"
)

type Generator struct {
	SchemasDir string
}

func NewGenerator(schemasDir string) *Generator {
	if schemasDir == "" {
		schemasDir = "schemas"
	}
	return &Generator{SchemasDir: schemasDir}
}

func (g *Generator) Generate(api API) (*mcp.Tool, error) {
	// For Phase 2, we'll use a simplified generation logic.
	// In a real scenario, this might probe the API or use OpenAPI.
	
	tool := &mcp.Tool{
		Name:        api.Name, // We can refine this later
		Description: api.Description,
		Module:      api.Module,
		InputSchema: mcp.InputSchema{
			Type: "object",
			Properties: make(map[string]mcp.Property),
		},
		Endpoint: &mcp.Endpoint{
			Method: api.Method,
			Path:   api.URL,
			Auth: mcp.AuthInfo{
				Type:     api.AuthType,
				Header:   api.AuthHeader,
				KeyRef:   api.AuthKey,
				Username: api.AuthUsername,
				Token:    api.AuthToken,
			},
		},
	}

	// Add some default params for GET requests
	if api.Method == "GET" {
		tool.InputSchema.Properties["page"] = mcp.Property{
			Type:        "integer",
			Description: "Page number for pagination",
			Default:     1,
		}
	}

	return tool, g.Save(tool)
}

func (g *Generator) Save(tool *mcp.Tool) error {
	dir := filepath.Join(g.SchemasDir, tool.Module)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, tool.Name+".json")
	data, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
