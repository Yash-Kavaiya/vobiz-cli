package numbers

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newBuyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "buy <number>",
		Short: "Buy a phone number from inventory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runBuy(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runBuy(api client.NumbersAPI, w io.Writer, number string) error {
	if err := api.Buy(context.Background(), number); err != nil {
		return err
	}
	fmt.Fprintf(w, "Purchased %s.\n", number)
	return nil
}
