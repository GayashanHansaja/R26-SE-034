// internal/cli/cache.go
package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage semantic cache",
	Long: `The cache command provides tools to monitor and manage the middleware's 
two-layer (Exact + Semantic) caching system. You can view real-time statistics 
and manually flush entries by tool, module, or for the entire system.`,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache hit/miss rates and memory usage",
	Long: `Display high-level statistics for the semantic cache, 
including total key counts, memory usage in Redis, and hit/miss trends.`,
	Example: `  bridgectl cache stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, ok := cfg.Contexts[cfg.CurrentContext]
		if !ok {
			return NewError(CodePrecondFail, "NO_CONTEXT", 
				"No context selected", 
				"Use 'bridgectl context set' to select an active environment.")
		}

		resp, err := http.Get(ctx.Server + "/api/cache/stats")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return NewError(CodeGeneralErr, "SERVER_ERROR", 
				fmt.Sprintf("Server returned error: %s", resp.Status), 
				"Verify the middleware server is running and reachable.")
		}

		var result any
		return formatter.Print(output.NewRawResponse(resp.Body, &result))
	},
}

var (
	flushModule string
	flushAll    bool
)

var cacheFlushCmd = &cobra.Command{
	Use:   "flush [tool]",
	Short: "Delete cache entries",
	Long: `Manually invalidate cache entries stored in Redis. 
You can target a specific tool by name, an entire module using the --module flag, 
or clear the entire cache with --all.`,
	Example: `  bridgectl cache flush finance.get-invoices
  bridgectl cache flush --module hr
  bridgectl cache flush --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, ok := cfg.Contexts[cfg.CurrentContext]
		if !ok {
			return NewError(CodePrecondFail, "NO_CONTEXT", 
				"No context selected", 
				"Use 'bridgectl context set' to select an active environment.")
		}

		u, _ := url.Parse(ctx.Server + "/api/cache/flush")
		q := u.Query()
		if len(args) > 0 {
			q.Set("tool", args[0])
		}
		if flushModule != "" {
			q.Set("module", flushModule)
		}
		if flushAll {
			q.Set("all", "true")
		}
		u.RawQuery = q.Encode()

		resp, err := http.Get(u.String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var result FlushResponse
		return formatter.Print(output.NewRawResponse(resp.Body, &result))
	},
}

type FlushResponse struct {
	Deleted int    `json:"deleted"`
	Status  string `json:"status"`
}

func (r *FlushResponse) RenderTable(w io.Writer) error {
	fmt.Fprintf(w, "Deleted %d cache entries.\n", r.Deleted)
	return nil
}

func init() {
	RootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheFlushCmd)

	cacheFlushCmd.Flags().StringVarP(&flushModule, "module", "m", "", "Flush entire module")
	cacheFlushCmd.Flags().BoolVarP(&flushAll, "all", "a", false, "Flush everything")
}
