package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimendra/ERPBridge/internal/idp"
	"github.com/nimendra/ERPBridge/internal/mcp"
	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage MCP tool schemas",
	Long: `The tool command provides utilities to bridge the gap between raw ERP APIs 
and the Model Context Protocol (MCP). It includes generators to transform 
API definitions or OpenAPI specs into MCP Tool schemas, and validators to 
ensure those schemas are ready for AI agent consumption.`,
}

var toolGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Auto-generate an MCP tool schema from a registered API or OpenAPI spec",
	Long: `Create a protocol-compliant MCP tool definition automatically. 
You can generate a tool from an API already registered in bridgectl, 
or point directly to an OpenAPI YAML/JSON file to batch-generate 
definitions for all operations.`,
	Example: `  bridgectl tool generate --api get-invoices
  bridgectl tool generate --api my-erp --openapi ./specs/erp.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiName, _ := cmd.Flags().GetString("api")
		openapiURL, _ := cmd.Flags().GetString("openapi")

		reg, err := idp.NewRegistry("", RootLog)
		if err != nil {
			return err
		}

		api, ok := reg.Get(apiName)
		if !ok {
			return fmt.Errorf("API %s not found", apiName)
		}

		gen := idp.NewGenerator("", RootLog)

		if openapiURL != "" {
			tools, err := gen.GenerateFromOpenAPI(api, openapiURL)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Generated %d tools from OpenAPI spec.\n", len(tools))
			return nil
		}

		tool, err := gen.Generate(api)
		if err != nil {
			return err
		}

		resp := &ToolGenerateResponse{
			ToolName: tool.Name,
			Module:   tool.Module,
			Path:     fmt.Sprintf("schemas/%s/%s.json", tool.Module, tool.Name),
		}

		return formatter.Print(resp)
	},
}

type ToolGenerateResponse struct {
	ToolName string `json:"toolName"`
	Module   string `json:"module"`
	Path     string `json:"path"`
}

func (r *ToolGenerateResponse) RenderTable(w io.Writer) error {
	_, _ = fmt.Fprintf(w, "Generating tool schema for  %s\n\n", r.ToolName)
	_, _ = fmt.Fprintf(w, "  Saving schema...       ✓ %s\n\n", r.Path)
	_, _ = fmt.Fprintf(w, "Tool name    %s\n", r.ToolName)
	_, _ = fmt.Fprintln(w, "\n✓ Tool generated successfully.")
	return nil
}

var toolListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all generated MCP tool schemas",
	Long:    `Scan the schemas/ directory and display all available MCP tool definitions.`,
	Example: `  bridgectl tool list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schemasDir := "schemas"
		var items []ToolListItem

		err := filepath.Walk(schemasDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".json" {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				var t mcp.Tool
				if err := json.Unmarshal(data, &t); err != nil {
					return nil
				}
				items = append(items, ToolListItem{
					Name:        t.Name,
					Module:      t.Module,
					Status:      "active",
					GeneratedAt: info.ModTime(),
				})
			}
			return nil
		})
		if err != nil {
			return err
		}

		resp := &ToolListResponse{Items: items, Total: len(items)}
		return formatter.Print(resp)
	},
}

type ToolListItem struct {
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Status      string    `json:"status"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type ToolListResponse struct {
	Items []ToolListItem `json:"items"`
	Total int            `json:"total"`
}

func (r *ToolListResponse) RenderTable(w io.Writer) error {
	tw := output.NewTabWriter(w)
	_, _ = fmt.Fprintln(tw, "NAME\tMODULE\tSTATUS\tGENERATED")
	for _, item := range r.Items {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			item.Name, item.Module, item.Status,
			item.GeneratedAt.Format("2006-01-02 15:04"))
	}
	return tw.Flush()
}

var toolInvokeCmd = &cobra.Command{
	Use:   "invoke [name] [arguments]",
	Short: "Invoke an MCP tool directly",
	Long: `Perform a direct invocation of an MCP tool through the middleware's 
internal API. This bypasses the full MCP transport but follows the same 
logic (including caching and validation), making it ideal for testing 
how an AI agent would experience the tool.`,
	Example: `  bridgectl tool invoke finance.get-invoices '{"page": 1}'`,
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		var argMap map[string]any
		if len(args) > 1 {
			if err := json.Unmarshal([]byte(args[1]), &argMap); err != nil {
				return NewError(CodeBadArgs, "INVALID_ARGUMENTS",
					fmt.Sprintf("Invalid arguments JSON: %v", err),
					"Ensure the arguments are a valid JSON string, e.g., '{\"page\": 1}'.")
			}
		}

		ctx := cfg.ActiveContext()
		mcpURL := ctx.MCPServer + "/api/tools/invoke"

		reqBody, _ := json.Marshal(mcp.ToolCallRequest{
			Name:      name,
			Arguments: argMap,
		})

		// Use manual HTTP client for direct invoke
		var netClient = &http.Client{
			Timeout: time.Second * 10,
		}

		// In a real app, use the context override
		resp, err := netClient.Post(mcpURL, "application/json", strings.NewReader(string(reqBody)))
		if err != nil {
			return fmt.Errorf("MCP server call failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var toolResult mcp.ToolResult
		if err := json.NewDecoder(resp.Body).Decode(&toolResult); err != nil {
			return fmt.Errorf("decode result failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Invoking  %s\n", name)
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "Args      %s\n\n", args[1])
		}

		return formatter.Print(toolResult)
	},
}

var toolValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate an MCP tool schema (JSON) or an OpenAPI spec (YAML)",
	Long: `Pre-flight check for schema files. This command validates that 
a JSON schema follows the MCP tool structure or that an OpenAPI 
specification can be correctly parsed and transformed by ERPBridge.`,
	Example: `  bridgectl tool validate schemas/finance/get-invoices.json
  bridgectl tool validate ./specs/legacy-erp.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		ext := filepath.Ext(path)

		if ext == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return NewError(CodeNotFound, "FILE_NOT_FOUND",
					fmt.Sprintf("Schema file not found: %s", path),
					"Check the file path and try again.")
			}
			var tool mcp.Tool
			if err := json.Unmarshal(data, &tool); err != nil {
				return NewError(CodeBadArgs, "INVALID_SCHEMA",
					fmt.Sprintf("Invalid MCP tool schema: %v", err),
					"Ensure the file contains a valid MCP Tool JSON schema.")
			}
			if tool.Name == "" {
				return NewError(CodeBadArgs, "MISSING_TOOL_NAME",
					"Invalid schema: missing 'name' field",
					"MCP tool schemas must have a 'name' field.")
			}
			fmt.Fprintf(os.Stderr, "✓ MCP Tool schema '%s' is valid.\n", path)
			return nil
		}

		if ext == ".yaml" || ext == ".yml" {
			gen := idp.NewGenerator("", RootLog)
			// Mock API for validation context
			mockAPI := idp.API{Name: "validate", Module: "test"}
			_, err := gen.GenerateFromOpenAPI(mockAPI, path)
			if err != nil {
				return NewError(CodeBadArgs, "INVALID_OPENAPI",
					fmt.Sprintf("Invalid OpenAPI spec: %v", err),
					"Ensure the file is a valid OpenAPI specification compatible with ERPBridge.")
			}
			fmt.Fprintf(os.Stderr, "✓ OpenAPI specification '%s' is valid and compatible with ERPBridge.\n", path)
			return nil
		}

		return fmt.Errorf("unsupported file extension: %s (expected .json, .yaml, or .yml)", ext)
	},
}

func init() {
	RootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolGenerateCmd)
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolInvokeCmd)
	toolCmd.AddCommand(toolValidateCmd)

	toolGenerateCmd.Flags().String("api", "", "Name of the registered API to generate from")
	toolGenerateCmd.Flags().String("openapi", "", "URL or path to an OpenAPI spec")
	_ = toolGenerateCmd.MarkFlagRequired("api")
}
