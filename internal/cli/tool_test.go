package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nimendra/ERPBridge/internal/config"
	"github.com/nimendra/ERPBridge/internal/mcp"
	"github.com/nimendra/ERPBridge/internal/output"
	"github.com/stretchr/testify/require"
)

func TestToolListResponse_RenderTable(t *testing.T) {
	resp := &ToolListResponse{
		Tools: []*mcp.Tool{
			{
				Metadata: mcp.Metadata{
					Name:    "tool1",
					Module:  "hr",
					Version: "1.0",
					Status:  "active",
				},
			},
		},
	}
	var buf bytes.Buffer
	err := resp.RenderTable(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("tool1")) {
		t.Errorf("expected output to contain 'tool1'")
	}
}

func TestToolGetCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"metadata":{"name":"tool1","version":"1.0"}}]`))
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {MCPServer: ts.URL},
		},
	}
	var buf bytes.Buffer
	formatter = &output.Formatter{Format: output.FormatJSON, Out: &buf}

	err := toolGetCmd.RunE(toolGetCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("tool1")) {
		t.Errorf("expected output to contain tool1")
	}
}

func TestToolDeleteCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {MCPServer: ts.URL},
		},
	}

	toolDeleteCmd.SetContext(context.Background())
	err := toolDeleteCmd.RunE(toolDeleteCmd, []string{"tool1", "1.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolValidateCmd(t *testing.T) {
	content := `{"metadata":{"name":"t","version":"1"}}`
	err := os.WriteFile("test_tool.json", []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove("test_tool.json") }()

	require.NoError(t, toolValidateCmd.Flags().Set("file", "test_tool.json"))
	err = toolValidateCmd.RunE(toolValidateCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
