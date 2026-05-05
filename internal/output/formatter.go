package output

import (
	"encoding/json"
	"io"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

type Formatter struct {
	Format Format
	Out    io.Writer // defaults to os.Stdout
}

// Print serialises v in the configured format.
// For table format, v must implement TableRenderer.
func (f *Formatter) Print(v any) error {
	switch f.Format {
	case FormatJSON:
		enc := json.NewEncoder(f.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)

	case FormatYAML:
		return yaml.NewEncoder(f.Out).Encode(v)

	default: // table
		if tr, ok := v.(TableRenderer); ok {
			return tr.RenderTable(f.Out)
		}
		// Fallback: JSON for types that don't implement table rendering
		return json.NewEncoder(f.Out).Encode(v)
	}
}

// TableRenderer is implemented by any type that knows how to
// render itself as an aligned table.
type TableRenderer interface {
	RenderTable(w io.Writer) error
}

// NewTabWriter returns a pre-configured tab writer for table output.
func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
}
