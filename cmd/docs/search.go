package docs

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search Vobiz docs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(Factory(), strings.Join(args, " "), cmd.OutOrStdout())
		},
	}
}

func runSearch(m MCP, query string, w io.Writer) error {
	results, err := m.Search(context.Background(), query)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(w, "No results.")
		return nil
	}
	for _, r := range results {
		fmt.Fprintf(w, "• %s  [%s]\n  %s\n\n", r.Title, r.Path, r.Snippet)
	}
	return nil
}
