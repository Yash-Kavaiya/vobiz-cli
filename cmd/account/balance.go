package account

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newBalanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show current account balance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runBalance(a, cmd.OutOrStdout())
		},
	}
}

func runBalance(api client.AccountAPI, w io.Writer) error {
	b, err := api.Balance(context.Background())
	if err != nil {
		return err
	}
	fmt.Fprintln(w, b)
	return nil
}
