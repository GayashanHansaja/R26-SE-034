package cli

import (
	"io"
	"log/slog"
	"testing"

	"github.com/nimendra/ERPBridge/internal/config"
	"github.com/nimendra/ERPBridge/internal/output"
)

func setupTest() {
	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Server: "http://test"},
		},
	}
	formatter = &output.Formatter{
		Format: output.FormatTable,
		Out:    io.Discard,
	}
	RootLog = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandleError(t *testing.T) {
	// coverage
}

func TestExecute(t *testing.T) {
	// Execute() directly terminates the process on error, so we cannot test it easily without mocking os.Exit
	// We'll skip for now and focus on RunE of commands
}
