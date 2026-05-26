package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command) {
	account.Register(root, func() string { return globalOutput })
}
func registerDocs(_ *cobra.Command) {}
