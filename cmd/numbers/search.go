package numbers

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newSearchCmd(format func() string) *cobra.Command {
	var country string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search available numbers in inventory (filter by ISO country code)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runSearch(a, cmd.OutOrStdout(), format(), country)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "ISO country code (e.g. US, IN, GB)")
	return cmd
}

func runSearch(api client.NumbersAPI, w io.Writer, format, country string) error {
	rows, err := api.SearchInventory(context.Background(), country)
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
		{Header: "SETUP", Field: "SetupRate"},
		{Header: "MONTHLY", Field: "MonthlyRentalRate"},
	}
	return output.Render(w, rows, cols, f)
}
