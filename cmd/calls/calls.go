// Package calls implements `vobiz calls …` subcommands.
package calls

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

var Overrides runtime.Overrides

var CallsFactory = func() (client.CallsAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Calls, nil
}

func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "calls",
		Short: "Make outbound calls and inspect call records",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newMakeCmd())
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newGetCmd(format))
	parent.AddCommand(cmd)
}
