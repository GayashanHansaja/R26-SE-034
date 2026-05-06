package idp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/nimendra/ERPBridge/internal/mcp"
)

type Generator struct {
	SchemasDir string
	log        *slog.Logger
}

func NewGenerator(schemasDir string, rootLog *slog.Logger) *Generator {
	if schemasDir == "" {
		schemasDir = "schemas"
	}
	return &Generator{
		SchemasDir: schemasDir,
		log:        logger.Component(rootLog, "idp"),
	}
}

func (g *Generator) Generate(api API) (*mcp.Tool, error) {
	tool := &mcp.Tool{
		Name:        api.Name,
		Description: api.Description,
		Module:      api.Module,
		InputSchema: mcp.InputSchema{
			Type:       "object",
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

	if api.Method == "GET" {
		tool.InputSchema.Properties["page"] = mcp.Property{
			Type:        "integer",
			Description: "Page number for pagination",
			Default:     1,
		}
	}

	g.log.Info("tool generated", slog.String("tool_name", tool.Name))
	return tool, g.Save(tool)
}

// Task 2: OpenAPI Integration
func (g *Generator) GenerateFromOpenAPI(api API, openapiURL string) ([]*mcp.Tool, error) {
	loader := openapi3.NewLoader()
	var doc *openapi3.T
	var err error

	if strings.HasPrefix(openapiURL, "http") {
		u, err := url.Parse(openapiURL)
		if err != nil {
			return nil, fmt.Errorf("invalid openapi url: %w", err)
		}
		doc, err = loader.LoadFromURI(u)
		if err != nil {
			// Fallback: manually fetch if LoadFromURI fails
			resp, httpErr := http.Get(openapiURL)
			if httpErr != nil {
				return nil, fmt.Errorf("failed to fetch openapi spec: %w", httpErr)
			}
			defer func() { _ = resp.Body.Close() }()
			doc, err = loader.LoadFromIoReader(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to parse fetched openapi spec: %w", err)
			}
		}
	} else {
		doc, err = loader.LoadFromFile(openapiURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load openapi spec: %w", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid openapi spec: %w", err)
	}

	var tools []*mcp.Tool
	for path, pathItem := range doc.Paths.Map() {
		for method, op := range pathItem.Operations() {
			toolName := fmt.Sprintf("%s.%s", api.Module, op.OperationID)
			if op.OperationID == "" {
				// Sanitize path for tool name
				safePath := strings.ReplaceAll(strings.Trim(path, "/"), "/", "-")
				toolName = fmt.Sprintf("%s.%s-%s", api.Module, method, safePath)
			}

			baseURL := ""
			if len(doc.Servers) > 0 {
				baseURL = doc.Servers[0].URL
			}

			tool := &mcp.Tool{
				Name:        toolName,
				Description: op.Summary,
				Module:      api.Module,
				InputSchema: mcp.InputSchema{
					Type:       "object",
					Properties: make(map[string]mcp.Property),
				},
				Endpoint: &mcp.Endpoint{
					Method: method,
					Path:   baseURL + path,
					Auth: mcp.AuthInfo{
						Type:     api.AuthType,
						Header:   api.AuthHeader,
						KeyRef:   api.AuthKey,
						Username: api.AuthUsername,
						Token:    api.AuthToken,
					},
				},
			}

			if tool.Description == "" {
				tool.Description = op.Description
			}

			// Map parameters
			for _, paramRef := range op.Parameters {
				param := paramRef.Value
				if param == nil || param.Schema == nil {
					continue
				}
				p := mcp.Property{
					Type:        "string", // Default
					Description: param.Description,
				}
				if len(param.Schema.Value.Type.Slice()) > 0 {
					p.Type = param.Schema.Value.Type.Slice()[0]
				}
				if param.Required {
					tool.InputSchema.Required = append(tool.InputSchema.Required, param.Name)
				}
				tool.InputSchema.Properties[param.Name] = p
			}

			// Map Request Body (for POST/PATCH)
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				content := op.RequestBody.Value.Content.Get("application/json")
				if content != nil && content.Schema != nil {
					schema := content.Schema.Value
					for propName, propRef := range schema.Properties {
						prop := propRef.Value
						p := mcp.Property{
							Type:        "string",
							Description: prop.Description,
						}
						if len(prop.Type.Slice()) > 0 {
							p.Type = prop.Type.Slice()[0]
						}
						tool.InputSchema.Properties[propName] = p
					}
					tool.InputSchema.Required = append(tool.InputSchema.Required, schema.Required...)
				}
			}

			// Task 6: Response Validation (infer output schema)
			resp200 := op.Responses.Status(200)
			if resp200 == nil {
				resp200 = op.Responses.Status(201)
			}
			if resp200 != nil && resp200.Value != nil && resp200.Value.Content.Get("application/json") != nil {
				schema := resp200.Value.Content.Get("application/json").Schema.Value
				var outputSchema any = schema
				tool.OutputSchema = &outputSchema
			}

			if err := g.Save(tool); err != nil {
				return nil, err
			}
			g.log.Info("tool generated from OpenAPI", slog.String("tool_name", toolName))
			tools = append(tools, tool)
		}
	}

	return tools, nil
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
