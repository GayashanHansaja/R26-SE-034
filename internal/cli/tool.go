package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nimendra/ERPBridge/internal/idp"
	"github.com/nimendra/ERPBridge/internal/mcp"
	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage MCP tool schemas",
}

var toolGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Auto-generate an MCP tool schema from a registered API",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiName, _ := cmd.Flags().GetString("api")
		
		reg, err := idp.NewRegistry("")
		if err != nil {
			return err
		}

		api, ok := reg.Get(apiName)
		if !ok {
			return fmt.Errorf("API %s not found", apiName)
		}

		gen := idp.NewGenerator("")
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
	fmt.Fprintf(w, "Generating tool schema for  %s\n\n", r.ToolName)
	fmt.Fprintf(w, "  Saving schema...       ✓ %s\n\n", r.Path)
	fmt.Fprintf(w, "Tool name    %s\n", r.ToolName)
	fmt.Fprintln(w, "\n✓ Tool generated successfully.")
	return nil
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all generated MCP tool schemas",
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
	fmt.Fprintln(tw, "NAME\tMODULE\tSTATUS\tGENERATED")
	for _, item := range r.Items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			item.Name, item.Module, item.Status,
			item.GeneratedAt.Format("2006-01-02 15:04"))
	}
	return tw.Flush()
}

var toolInvokeCmd = &cobra.Command{
	Use:   "invoke [name] [arguments]",
	Short: "Invoke an MCP tool directly",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		var argMap map[string]any
		if len(args) > 1 {
			if err := json.Unmarshal([]byte(args[1]), &argMap); err != nil {
				return fmt.Errorf("invalid arguments JSON: %w", err)
			}
		}

		ctx := cfg.ActiveContext()
		mcpURL := ctx.MCPServer + "/api/tools/invoke"

		reqBody, _ := json.Marshal(mcp.ToolCallRequest{
			Name:      name,
			Arguments: argMap,
		})

		resp, err := http.Post(mcpURL, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return fmt.Errorf("MCP server call failed: %w", err)
		}
		defer resp.Body.Close()

		var toolResult mcp.ToolResult
		if err := json.NewDecoder(resp.Body).Decode(&toolResult); err != nil {
			return fmt.Errorf("decode result failed: %w", err)
		}

		fmt.Fprintf(os.Stdout, "Invoking  %s\n", name)
		fmt.Fprintf(os.Stdout, "Args      %s\n\n", args[1])

		return formatter.Print(toolResult)
	},
}

func init() {
	RootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolGenerateCmd)
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolInvokeCmd)

	toolGenerateCmd.Flags().String("api", "", "Name of the registered API to generate from")
	toolGenerateCmd.MarkFlagRequired("api")
}
