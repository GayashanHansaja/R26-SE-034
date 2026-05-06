package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nimendra/ERPBridge/internal/config"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	outputFormat string
	ctxOverride  string
	verbose     bool

	cfg       *config.Config
	formatter *output.Formatter
	RootLog   *slog.Logger
)

var RootCmd = &cobra.Command{
	Use:     "bridgectl",
	Short:   "Middleware for Bridging Legacy ERP and Agentic AI",
	Version: "1.0.0",
	Long: `bridgectl is the developer CLI for the ERPBridge ecosystem. 
It provides tools to manage environments, register and test ERP APIs, 
generate and validate MCP tool schemas, and monitor the middleware's 
health through real-time log streaming and cache analytics.

The CLI interacts with the ERPBridge middleware via a REST API 
and supports multiple output formats including Table, JSON, and YAML.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Logger for CLI
		if verbose {
			os.Setenv("LOG_LEVEL", "debug")
		} else {
			os.Setenv("LOG_LEVEL", "error") // Only errors in CLI by default
		}
		RootLog = logger.Init()

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if ctxOverride != "" {
			cfg.CurrentContext = ctxOverride
		}

		formatter = &output.Formatter{
			Format: output.Format(outputFormat),
			Out:    os.Stdout,
		}
		return nil
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json, yaml")
	RootCmd.PersistentFlags().StringVarP(&ctxOverride, "context", "c", "", "Override active context")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show full HTTP request/response detail")
}
