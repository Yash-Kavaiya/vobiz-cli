package numbers

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
	"github.com/yash-kavaiya/vobiz-cli/internal/paginate"
)

func newListCmd(format func() string) *cobra.Command {
	var (
		limit int
		all   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List owned phone numbers (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runList(a, cmd.OutOrStdout(), format(), limit, all)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max number of rows")
	cmd.Flags().BoolVar(&all, "all", false, "fetch all pages")
	return cmd
}

func runList(api client.NumbersAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Number], error) {
		items, next, err := api.List(ctx, cursor)
		if err != nil {
			return paginate.Page[client.Number]{}, err
		}
		return paginate.Page[client.Number]{Items: items, NextCursor: next}, nil
	}
	var (
		rows []client.Number
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
		{Header: "NUMBER", Field: "Number"},
		{Header: "COUNTRY", Field: "Country"},
		{Header: "TYPE", Field: "NumberType"},
		{Header: "MONTHLY", Field: "MonthlyRentalRate"},
		{Header: "APPLICATION", Field: "Application"},
	}
	return output.Render(w, rows, cols, f)
}
