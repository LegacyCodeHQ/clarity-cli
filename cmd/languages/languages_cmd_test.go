package languages

import (
	"bytes"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/LegacyCodeHQ/sanity/depgraph"
)

func TestLanguagesCommand_PrintsSupportedLanguagesAndExtensions(t *testing.T) {
	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}

	var expected strings.Builder
	writer := tabwriter.NewWriter(&expected, 0, 0, 2, ' ', 0)
	_, _ = writer.Write([]byte("Language\tMaturity\tExtensions\n"))
	_, _ = writer.Write([]byte("--------\t--------\t----------\n"))
	for _, language := range depgraph.SupportedLanguages() {
		_, _ = writer.Write([]byte(language.Name))
		_, _ = writer.Write([]byte("\t"))
		_, _ = writer.Write([]byte(language.Maturity.String()))
		_, _ = writer.Write([]byte("\t"))
		_, _ = writer.Write([]byte(strings.Join(language.Extensions, ", ")))
		_, _ = writer.Write([]byte("\n"))
	}
	_ = writer.Flush()

	if out.String() != expected.String() {
		t.Fatalf("output = %q, want %q", out.String(), expected.String())
	}
}
