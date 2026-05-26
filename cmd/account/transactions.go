package account

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
	"github.com/yash-kavaiya/vobiz-cli/internal/paginate"
)

func newTransactionsCmd(format func() string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "List account transactions",
	}
	var (
		limit int
		all   bool
	)
	list := &cobra.Command{
		Use:   "list",
		Short: "List transactions (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runTransactions(a, cmd.OutOrStdout(), format(), limit, all)
		},
	}
	list.Flags().IntVar(&limit, "limit", 50, "max number of rows")
	list.Flags().BoolVar(&all, "all", false, "fetch all pages")
	cmd.AddCommand(list)
	return cmd
}

func runTransactions(api client.AccountAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Transaction], error) {
		items, next, err := api.Transactions(ctx, cursor, limit)
		if err != nil {
			return paginate.Page[client.Transaction]{}, err
		}
		return paginate.Page[client.Transaction]{Items: items, NextCursor: next}, nil
	}

	var (
		rows []client.Transaction
		err  error
	)
	if all {
		rows, err = paginate.All(context.Background(), fetch)
	} else {
		rows, err = paginate.AllN(context.Background(), fetch, limit)
	}
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "ID", Field: "ID"},
		{Header: "AMOUNT", Field: "Amount"},
		{Header: "DESCRIPTION", Field: "Description"},
		{Header: "DATE", Field: "CreatedAt"},
	}
	return output.Render(w, rows, cols, f)
}
