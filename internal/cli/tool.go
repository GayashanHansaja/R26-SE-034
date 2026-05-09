package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nimendra/ERPBridge/internal/idp"
	"github.com/nimendra/ERPBridge/internal/mcp"
	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage MCP tool resources (V2 Control Plane)",
}

var toolApplyCmd = &cobra.Command{
	Use:   "apply -f [file]",
	Short: "Apply a tool schema to the registry (declarative)",
	Example: `  bridgectl tool apply -f list_employees.yaml
  bridgectl tool apply -f schemas/hr/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return fmt.Errorf("file path is required")
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		var tool mcp.Tool
		if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
			if err := yaml.Unmarshal(data, &tool); err != nil {
				return fmt.Errorf("unmarshal yaml: %w", err)
			}
		} else {
			if err := json.Unmarshal(data, &tool); err != nil {
				return fmt.Errorf("unmarshal json: %w", err)
			}
		}

		ctx := cfg.ActiveContext()
		if err := ValidateServerURL(ctx.MCPServer, "MCP", cfg.CurrentContext); err != nil {
			return err
		}
		url := ctx.MCPServer + "/apis/erpbridge.io/v1/tools"

		payload, _ := json.Marshal(tool)
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "tool %s@%s applied successfully\n", tool.Metadata.Name, tool.Metadata.Version)
		return nil
	},
}

var toolGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Display one or many tool resources",
	Example: `  bridgectl tool get
  bridgectl tool get list_employees -o yaml`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ctx := cfg.ActiveContext()
		if err := ValidateServerURL(ctx.MCPServer, "MCP", cfg.CurrentContext); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		url := ctx.MCPServer + "/apis/erpbridge.io/v1/tools"

		resp, err := http.Get(url)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		defer func() { _ = resp.Body.Close() }()

		var tools []*mcp.Tool
		if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, t := range tools {
			if strings.HasPrefix(t.Metadata.Name, toComplete) {
				completions = append(completions, t.Metadata.Name)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cfg.ActiveContext()
		if err := ValidateServerURL(ctx.MCPServer, "MCP", cfg.CurrentContext); err != nil {
			return err
		}
		url := ctx.MCPServer + "/apis/erpbridge.io/v1/tools"

		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		var tools []*mcp.Tool
		if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")

		if len(args) > 0 {
			name, version := mcp.ParseToolIdentifier(args[0])
			var target *mcp.Tool
			for _, t := range tools {
				if t.Metadata.Name == name && (version == "" || t.Metadata.Version == version) {
					target = t
					break
				}
			}

			if target == nil {
				return fmt.Errorf("tool %s not found", args[0])
			}

			switch outputFormat {
			case "yaml":
				y, _ := yaml.Marshal(target)
				fmt.Println(string(y))
				return nil
			case "json":
				j, _ := json.MarshalIndent(target, "", "  ")
				fmt.Println(string(j))
				return nil
			}
			tools = []*mcp.Tool{target}
		}

		res := &ToolListResponse{Tools: tools}
		return formatter.Print(res)
	},
}

type ToolListResponse struct {
	Tools []*mcp.Tool `json:"tools"`
}

func (r *ToolListResponse) RenderTable(w io.Writer) error {
	tw := output.NewTabWriter(w)
	_, _ = fmt.Fprintln(tw, "NAME\tMODULE\tVERSION\tSTATUS")
	for _, t := range r.Tools {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			t.Metadata.Name, t.Metadata.Module, t.Metadata.Version, t.Metadata.Status)
	}
	return tw.Flush()
}

var toolDescribeCmd = &cobra.Command{
	Use:   "describe [name]",
	Short: "Show details of a specific tool resource",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return toolGetCmd.ValidArgsFunction(cmd, args, toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// For now, reuse GET with rich formatting or implement separate describe logic
		fmt.Printf("Describing tool: %s (Not fully implemented, use 'get -o yaml' for now)\n", args[0])
		return nil
	},
}

var toolValidateCmd = &cobra.Command{
	Use:   "validate -f [file]",
	Short: "Locally validate a tool schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		var tool mcp.Tool
		if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
			_ = yaml.Unmarshal(data, &tool)
		} else {
			_ = json.Unmarshal(data, &tool)
		}

		if tool.Metadata.Name == "" {
			return fmt.Errorf("validation failed: metadata.name is missing")
		}
		if tool.Metadata.Version == "" {
			return fmt.Errorf("validation failed: metadata.version is missing")
		}

		fmt.Printf("✓ tool %s@%s is locally valid\n", tool.Metadata.Name, tool.Metadata.Version)
		return nil
	},
}

var toolGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Auto-generate an MCP tool schema from a registered API or OpenAPI spec",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiName, _ := cmd.Flags().GetString("api")
		openapiURL, _ := cmd.Flags().GetString("openapi")

		reg, err := idp.NewRegistry("", RootLog)
		if err != nil {
			return err
		}

		api, ok := reg.Get(apiName)
		if !ok {
			return NewError(CodeNotFound, "API_NOT_FOUND",
				fmt.Sprintf("API %q not found in local registry", apiName),
				"A registered API is required to provide base URL and authentication templates. Run 'bridgectl api register' first.")
		}

		gen := idp.NewGenerator("", RootLog)

		if openapiURL != "" {
			tools, err := gen.GenerateFromOpenAPI(cmd.Context(), api, openapiURL)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "generated %d tools from OpenAPI spec\n", len(tools))
			return nil
		}

		tool, err := gen.Generate(api)
		if err != nil {
			return err
		}

		fmt.Printf("Generated tool: %s\n", tool.Metadata.Name)
		return nil
	},
}

var toolDeleteCmd = &cobra.Command{
	Use:   "delete [name] [version]",
	Short: "Remove a tool from the registry",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		version := args[1]

		ctx := cfg.ActiveContext()
		if err := ValidateServerURL(ctx.MCPServer, "MCP", cfg.CurrentContext); err != nil {
			return err
		}
		url := fmt.Sprintf("%s/apis/erpbridge.io/v1/tools?name=%s&version=%s", ctx.MCPServer, name, version)

		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodDelete, url, nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(body))
		}

		fmt.Printf("tool %s@%s deleted\n", name, version)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolApplyCmd)
	toolCmd.AddCommand(toolGetCmd)
	toolCmd.AddCommand(toolDescribeCmd)
	toolCmd.AddCommand(toolValidateCmd)
	toolCmd.AddCommand(toolGenerateCmd)
	toolCmd.AddCommand(toolDeleteCmd)

	toolApplyCmd.Flags().StringP("file", "f", "", "Path to the tool schema file")
	toolGetCmd.Flags().StringP("output", "o", "table", "Output format (table|yaml|json)")
	toolValidateCmd.Flags().StringP("file", "f", "", "Path to the tool schema file")
	toolGenerateCmd.Flags().String("api", "", "Name of the registered API to generate from")
	toolGenerateCmd.Flags().String("openapi", "", "URL or path to an OpenAPI spec")
	_ = toolGenerateCmd.MarkFlagRequired("api")
}
