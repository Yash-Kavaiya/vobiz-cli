package calls

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newGetCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <call-uuid>",
		Short: "Get a single call detail record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := CallsFactory()
			if err != nil {
				return err
			}
			return runGet(a, cmd.OutOrStdout(), format(), args[0])
		},
	}
}

func runGet(api client.CallsAPI, w io.Writer, format, callID string) error {
	cdr, err := api.GetCDR(context.Background(), callID)
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "UUID", Field: "UUID"},
		{Header: "FROM", Field: "CallerIDNumber"},
		{Header: "TO", Field: "DestinationNumber"},
		{Header: "DIR", Field: "CallDirection"},
		{Header: "DURATION", Field: "Duration"},
		{Header: "BILLSEC", Field: "BillSec"},
		{Header: "COST", Field: "Cost"},
		{Header: "HANGUP", Field: "HangupCause"},
		{Header: "MOS", Field: "MOS"},
	}
	return output.Render(w, *cdr, cols, f)
}
