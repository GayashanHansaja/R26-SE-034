package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

type MockData struct {
	Name string `json:"name" yaml:"name"`
}

func (m *MockData) RenderTable(w io.Writer) error {
	tw := NewTabWriter(w)
	_, _ = fmt.Fprintf(tw, "NAME\n%s\n", m.Name)
	return tw.Flush()
}

func TestFormatter_Print(t *testing.T) {
	data := &MockData{Name: "Test"}

	t.Run("JSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		f := &Formatter{Format: FormatJSON, Out: buf}
		if err := f.Print(data); err != nil {
			t.Fatal(err)
		}
		expected := `{
  "name": "Test"
}
`
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Table", func(t *testing.T) {
		buf := &bytes.Buffer{}
		f := &Formatter{Format: FormatTable, Out: buf}
		if err := f.Print(data); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "NAME") || !strings.Contains(buf.String(), "Test") {
			t.Errorf("unexpected table output: %q", buf.String())
		}
	})
}
