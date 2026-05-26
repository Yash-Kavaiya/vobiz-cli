// Package auth implements `vobiz auth …` subcommands.
package auth

import "github.com/spf13/cobra"

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Vobiz credentials and profiles",
	}
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newProfileCmd())
	parent.AddCommand(cmd)
}
