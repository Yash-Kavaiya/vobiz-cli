// Package docs implements `vobiz docs …` subcommands.
package docs

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/docsmcp"
)

// MCP is the interface satisfied by both the real client and test fakes.
type MCP interface {
	Search(ctx context.Context, query string) ([]docsmcp.Result, error)
	Fetch(ctx context.Context, path string) (string, error)
}

// Factory builds the real client; replaced in tests if needed.
var Factory = func() MCP { return docsmcp.New("") }

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Search and read Vobiz documentation in your terminal",
	}
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newOpenCmd())
	parent.AddCommand(cmd)
}
