package auth

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active profile and stored Auth ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runStatus(path, cmd.OutOrStdout())
		},
	}
}

func runStatus(path string, out io.Writer) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	if f.ActiveProfile == "" {
		fmt.Fprintln(out, "No active profile. Run 'vobiz auth login'.")
		return nil
	}
	p := f.Profiles[f.ActiveProfile]
	fmt.Fprintf(out, "Active profile: %s\nAuth ID:        %s\nBase URL:       %s\nConfig file:    %s\n",
		f.ActiveProfile, p.AuthID, fallback(p.BaseURL, cliAuth.DefaultBaseURL), path)
	return nil
}

func fallback(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
