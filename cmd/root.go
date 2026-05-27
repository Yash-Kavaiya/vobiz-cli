// Package cmd wires the Cobra command tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flag-backing vars. Tests may read these to assert wiring.
var (
	globalOutput  string
	globalProfile string
	globalAuthID  string
	globalAuthTok string
	globalBaseURL string
	globalVerbose bool
	globalNoColor bool
)

// New constructs the root *cobra.Command. Subcommands are added by their
// respective packages' Register functions (see cmd/auth, cmd/account, cmd/docs).
func New() *cobra.Command {
	root := &cobra.Command{
		Use:           "vobiz",
		Short:         "Vobiz CLI — programmable telephony from your terminal",
		Long:          "The unofficial-but-friendly terminal interface for the Vobiz programmable-telephony platform.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&globalOutput, "output", "o", "table", "output format: table|json|yaml")
	pf.StringVar(&globalProfile, "profile", "", "named profile from ~/.vobiz/config.yaml")
	pf.StringVar(&globalAuthID, "auth-id", "", "override Auth ID (env VOBIZ_AUTH_ID)")
	pf.StringVar(&globalAuthTok, "auth-token", "", "override Auth Token (env VOBIZ_AUTH_TOKEN)")
	pf.StringVar(&globalBaseURL, "base-url", "", "override API base URL")
	pf.BoolVarP(&globalVerbose, "verbose", "v", false, "verbose output")
	pf.BoolVar(&globalNoColor, "no-color", false, "disable color output")

	registerVersion(root)
	registerCompletion(root)
	registerAuth(root)
	registerAccount(root)
	registerDocs(root)

	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	return root
}
