package docs

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <path>",
		Short: "Fetch and render a Vobiz docs page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpen(Factory(), args[0], cmd.OutOrStdout())
		},
	}
}

func runOpen(m MCP, path string, w io.Writer) error {
	md, err := m.Fetch(context.Background(), path)
	if err != nil {
		return err
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// fall back to raw markdown if glamour can't initialize
		fmt.Fprint(w, md)
		return nil
	}
	rendered, err := r.Render(md)
	if err != nil {
		fmt.Fprint(w, md)
		return nil
	}
	fmt.Fprint(w, rendered)
	return nil
}
