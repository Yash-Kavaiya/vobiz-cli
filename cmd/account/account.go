// Package account implements `vobiz account …` subcommands.
package account

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

// Overrides is populated by Register's PersistentPreRunE so that AccountFactory
// can see the global flag values at the time a subcommand runs.
var Overrides runtime.Overrides

var AccountFactory = func() (client.AccountAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Account, nil
}

// Register adds `account` and its children to the parent command.
// `format` returns the current value of the global -o flag; `ov` returns
// the current values of the global credential flags.
func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage your Vobiz account",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newBalanceCmd())
	cmd.AddCommand(newTransactionsCmd(format))
	cmd.AddCommand(newConcurrencyCmd(format))
	parent.AddCommand(cmd)
}
