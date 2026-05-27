package account

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newConcurrencyCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "concurrency",
		Short: "Show concurrent-call limits and current usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runConcurrency(a, cmd.OutOrStdout(), format())
		},
	}
}

func runConcurrency(api client.AccountAPI, w io.Writer, format string) error {
	c, err := api.Concurrency(context.Background())
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "LIMIT", Field: "Limit"},
		{Header: "CURRENT", Field: "Current"},
	}
	return output.Render(w, *c, cols, f)
}
