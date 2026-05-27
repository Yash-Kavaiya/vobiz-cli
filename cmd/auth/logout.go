package auth

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func newLogoutCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials for a profile (default: active profile)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runLogout(path, profile, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "profile to remove (default: active)")
	return cmd
}

func runLogout(path, profile string, out io.Writer) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	name := profile
	if name == "" {
		name = f.ActiveProfile
	}
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	delete(f.Profiles, name)

	if f.ActiveProfile == name {
		f.ActiveProfile = rotateActive(f.Profiles)
	}
	if err := config.Save(path, f); err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed profile %q.\n", name)
	return nil
}

// rotateActive picks a deterministic survivor (lexicographically first key)
// from the remaining profiles, or returns "" if none remain. Deterministic so
// that repeated logouts on the same config produce the same active profile.
func rotateActive(profiles map[string]config.Profile) string {
	if len(profiles) == 0 {
		return ""
	}
	names := make([]string, 0, len(profiles))
	for k := range profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return names[0]
}
