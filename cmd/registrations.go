package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
	"github.com/yash-kavaiya/vobiz-cli/cmd/docs"
	"github.com/yash-kavaiya/vobiz-cli/cmd/numbers"
	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command) {
	account.Register(root, func() string { return globalOutput }, ovFn)
}
func registerNumbers(root *cobra.Command) {
	numbers.Register(root, func() string { return globalOutput }, ovFn)
}
func registerDocs(root *cobra.Command) { docs.Register(root) }

func ovFn() runtime.Overrides {
	return runtime.Overrides{
		Profile:     globalProfile,
		FlagID:      globalAuthID,
		FlagToken:   globalAuthTok,
		FlagBaseURL: globalBaseURL,
	}
}
