// Package auth implements `vobiz auth …` subcommands.
package auth

import "github.com/spf13/cobra"

// Register adds `auth` and its children to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Vobiz credentials and profiles",
	}
	cmd.AddCommand(newLoginCmd())
	parent.AddCommand(cmd)
}
