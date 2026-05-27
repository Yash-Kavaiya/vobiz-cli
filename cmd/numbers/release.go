package numbers

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <number>",
		Short: "Release a number back to inventory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runRelease(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runRelease(api client.NumbersAPI, w io.Writer, number string) error {
	if err := api.Release(context.Background(), number); err != nil {
		return err
	}
	fmt.Fprintf(w, "Released %s.\n", number)
	return nil
}
