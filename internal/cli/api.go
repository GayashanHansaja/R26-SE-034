package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/nimendra/ERPBridge/internal/idp"
	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Manage ERP API endpoints",
}

var apiRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new ERP API endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := idp.NewRegistry("")
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		method, _ := cmd.Flags().GetString("method")
		module, _ := cmd.Flags().GetString("module")
		desc, _ := cmd.Flags().GetString("description")
		authType, _ := cmd.Flags().GetString("auth-type")
		authHeader, _ := cmd.Flags().GetString("auth-header")
		authKey, _ := cmd.Flags().GetString("auth-key")

		api := &idp.API{
			Name:        name,
			URL:         url,
			Method:      method,
			Module:      module,
			Description: desc,
			AuthType:    authType,
			AuthHeader:  authHeader,
			AuthKey:     authKey,
		}

		if err := reg.Register(api); err != nil {
			return err
		}

		// Wrap in a response struct for formatting
		resp := &APIRegistrationResponse{API: *api}
		return formatter.Print(resp)
	},
}

type APIRegistrationResponse struct {
	API idp.API `json:"api" yaml:"api"`
}

func (r *APIRegistrationResponse) RenderTable(w io.Writer) error {
	fmt.Fprintf(w, "Registered API  %s\n", r.API.Name)
	fmt.Fprintf(w, "ID              %s\n", r.API.ID)
	fmt.Fprintf(w, "Module          %s\n", r.API.Module)
	fmt.Fprintf(w, "Method          %s\n", r.API.Method)
	fmt.Fprintf(w, "URL             %s\n", r.API.URL)
	fmt.Fprintf(w, "Status          %s\n", r.API.Status)
	fmt.Fprintln(w, "\nNext: run \"bridgectl tool generate --api "+r.API.Name+"\" to create an MCP tool schema.")
	return nil
}

var apiListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered APIs",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := idp.NewRegistry("")
		if err != nil {
			return err
		}

		apis := reg.List()
		resp := &APIListResponse{Items: apis}
		return formatter.Print(resp)
	},
}

type APIListResponse struct {
	Items []idp.API `json:"items" yaml:"items"`
}

func (r *APIListResponse) RenderTable(w io.Writer) error {
	tw := output.NewTabWriter(w)
	fmt.Fprintln(tw, "ID\tNAME\tMODULE\tMETHOD\tSTATUS")
	for _, api := range r.Items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", api.ID, api.Name, api.Module, api.Method, api.Status)
	}
	return tw.Flush()
}

var apiTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Send a test request to a registered API",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := idp.NewRegistry("")
		if err != nil {
			return err
		}

		name := args[0]
		api, ok := reg.Get(name)
		if !ok {
			return fmt.Errorf("API %s not found", name)
		}

		client := connector.NewClient()
		ep := connector.EndpointConfig{
			Method:  api.Method,
			Path:    api.URL, // In this case, URL is absolute as per register
			BaseURL: "",      // Empty because Path is the full URL
			Auth: connector.AuthConfig{
				Type:   api.AuthType,
				Header: api.AuthHeader,
				Key:    api.AuthKey,
			},
		}

		start := time.Now()
		resp, err := client.Call(context.Background(), ep, nil, nil)
		latency := time.Since(start)

		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		var body any
		json.NewDecoder(resp.Body).Decode(&body)

		testResp := &APITestResponse{
			API:       api,
			Status:    resp.Status,
			Code:      resp.StatusCode,
			Latency:   latency,
			Response:  body,
			IsSuccess: resp.StatusCode >= 200 && resp.StatusCode < 300,
		}

		return formatter.Print(testResp)
	},
}

type APITestResponse struct {
	API       idp.API       `json:"api"`
	Status    string        `json:"status"`
	Code      int           `json:"code"`
	Latency   time.Duration `json:"latency"`
	Response  any           `json:"response"`
	IsSuccess bool          `json:"isSuccess"`
}

func (r *APITestResponse) RenderTable(w io.Writer) error {
	fmt.Fprintf(w, "Testing  %s\n", r.API.Name)
	fmt.Fprintf(w, "URL      %s %s\n", r.API.Method, r.API.URL)
	fmt.Fprintf(w, "Auth     %s (%s)\n\n", r.API.AuthType, r.API.AuthHeader)
	fmt.Fprintf(w, "Status   %s\n", r.Status)
	fmt.Fprintf(w, "Latency  %v\n\n", r.Latency)

	if r.IsSuccess {
		fmt.Fprintln(w, "✓ API is reachable and auth is valid.")
	} else {
		fmt.Fprintln(w, "✗ API test failed")
	}
	return nil
}

func init() {
	RootCmd.AddCommand(apiCmd)
	apiCmd.AddCommand(apiRegisterCmd)
	apiCmd.AddCommand(apiListCmd)
	apiCmd.AddCommand(apiTestCmd)

	apiRegisterCmd.Flags().String("name", "", "Unique name for this API")
	apiRegisterCmd.Flags().String("url", "", "Full URL of the ERP endpoint")
	apiRegisterCmd.Flags().String("method", "GET", "HTTP method")
	apiRegisterCmd.Flags().String("module", "", "ERP module")
	apiRegisterCmd.Flags().String("description", "", "Human-readable description")
	apiRegisterCmd.Flags().String("auth-type", "api-key", "Auth type")
	apiRegisterCmd.Flags().String("auth-header", "X-API-Key", "Auth header")
	apiRegisterCmd.Flags().String("auth-key", "", "Auth key")

	apiRegisterCmd.MarkFlagRequired("name")
	apiRegisterCmd.MarkFlagRequired("url")
	apiRegisterCmd.MarkFlagRequired("module")
	apiRegisterCmd.MarkFlagRequired("description")
}
