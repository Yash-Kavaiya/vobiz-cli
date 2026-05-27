package account

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newGetCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show account details",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runGet(a, cmd.OutOrStdout(), format())
		},
	}
}

func runGet(api client.AccountAPI, w io.Writer, format string) error {
	acc, err := api.Get(context.Background())
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "AUTH ID", Field: "AuthID"},
		{Header: "TYPE", Field: "AccountType"},
		{Header: "BILLING", Field: "BillingMode"},
		{Header: "CREDITS", Field: "CashCredits"},
		{Header: "TZ", Field: "Timezone"},
	}
	return output.Render(w, *acc, cols, f)
}
