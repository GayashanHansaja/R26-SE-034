package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
	"github.com/goccy/go-yaml"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage bridgectl contexts",
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		var items []ContextItem
		for name := range cfg.Contexts {
			current := name == cfg.CurrentContext
			items = append(items, ContextItem{
				Name:    name,
				Server:  cfg.Contexts[name].Server,
				Current: current,
			})
		}
		resp := &ContextListResponse{Items: items}
		return formatter.Print(resp)
	},
}

type ContextItem struct {
	Name    string `json:"name"`
	Server  string `json:"server"`
	Current bool   `json:"current"`
}

type ContextListResponse struct {
	Items []ContextItem `json:"items"`
}

func (r *ContextListResponse) RenderTable(w io.Writer) error {
	tw := output.NewTabWriter(w)
	fmt.Fprintln(tw, "NAME\tSERVER\tCURRENT")
	for _, item := range r.Items {
		curr := ""
		if item.Current {
			curr = "✓"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", item.Name, item.Server, curr)
	}
	return tw.Flush()
}

var contextSetCmd = &cobra.Command{
	Use:   "set [name]",
	Short: "Switch active context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if _, ok := cfg.Contexts[name]; !ok {
			return fmt.Errorf("context %s not found", name)
		}
		cfg.CurrentContext = name
		return saveConfig()
	},
}

func saveConfig() error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".bridgectl", "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func init() {
	RootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextSetCmd)
}
