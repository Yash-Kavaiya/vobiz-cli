package calls

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

type listFlags struct {
	Page          int
	PerPage       int
	FromNumber    string
	ToNumber      string
	StartDate     string
	EndDate       string
	CallDirection string
	MinDuration   int
}

func newListCmd(format func() string) *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List call detail records (CDRs)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := CallsFactory()
			if err != nil {
				return err
			}
			return runList(a, cmd.OutOrStdout(), format(), f)
		},
	}
	cmd.Flags().IntVar(&f.Page, "page", 1, "page number")
	cmd.Flags().IntVar(&f.PerPage, "per-page", 20, "rows per page (max 100)")
	cmd.Flags().StringVar(&f.FromNumber, "from", "", "filter by caller ID number")
	cmd.Flags().StringVar(&f.ToNumber, "to", "", "filter by destination number")
	cmd.Flags().StringVar(&f.StartDate, "start", "", "ISO date — only calls on or after")
	cmd.Flags().StringVar(&f.EndDate, "end", "", "ISO date — only calls on or before")
	cmd.Flags().StringVar(&f.CallDirection, "direction", "", "inbound | outbound")
	cmd.Flags().IntVar(&f.MinDuration, "min-duration", 0, "only calls longer than N seconds")
	return cmd
}

func runList(api client.CallsAPI, w io.Writer, format string, f listFlags) error {
	rows, _, err := api.ListCDR(context.Background(), client.CDRListOpts{
		Page: f.Page, PerPage: f.PerPage,
		FromNumber: f.FromNumber, ToNumber: f.ToNumber,
		StartDate: f.StartDate, EndDate: f.EndDate,
		CallDirection: f.CallDirection, MinDuration: f.MinDuration,
	})
	if err != nil {
		return err
	}
	fmtSel, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "UUID", Field: "UUID"},
		{Header: "FROM", Field: "CallerIDNumber"},
		{Header: "TO", Field: "DestinationNumber"},
		{Header: "DIR", Field: "CallDirection"},
		{Header: "DUR", Field: "Duration"},
		{Header: "BILL", Field: "BillSec"},
		{Header: "COST", Field: "Cost"},
		{Header: "HANGUP", Field: "HangupCause"},
	}
	return output.Render(w, rows, cols, fmtSel)
}
