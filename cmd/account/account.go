// Package account implements `vobiz account …` subcommands.
package account

import (
	"fmt"

	"github.com/spf13/cobra"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

// AccountFactory is replaced in tests; in production it builds a real client.
var AccountFactory = func() (client.AccountAPI, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	creds, err := cliAuth.Resolve(cliAuth.Inputs{Config: cfg})
	if err != nil {
		return nil, err
	}
	return client.New(creds).Account, nil
}

func mustAccount() client.AccountAPI {
	a, err := AccountFactory()
	if err != nil {
		panic(fmt.Sprintf("account factory: %v", err))
	}
	return a
}

// Register adds the `account` subtree.
func Register(parent *cobra.Command, format func() string) {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage your Vobiz account",
	}
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newBalanceCmd())
	cmd.AddCommand(newTransactionsCmd(format))
	cmd.AddCommand(newConcurrencyCmd(format))
	parent.AddCommand(cmd)
}
