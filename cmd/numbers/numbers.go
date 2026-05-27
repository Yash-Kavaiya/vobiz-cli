// Package numbers implements `vobiz numbers …` subcommands.
package numbers

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

var Overrides runtime.Overrides

var NumbersFactory = func() (client.NumbersAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Numbers, nil
}

func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "numbers",
		Short: "Manage owned phone numbers and search inventory",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newSearchCmd(format))
	cmd.AddCommand(newBuyCmd())
	cmd.AddCommand(newReleaseCmd())
	parent.AddCommand(cmd)
}
