package auth

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "List, switch, or remove named profiles",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runProfileList(path, cmd.OutOrStdout())
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runProfileUse(path, args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a named profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runProfileRm(path, args[0])
		},
	})
	return cmd
}

func runProfileList(path string, out io.Writer) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(f.Profiles))
	for k := range f.Profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		marker := " "
		if n == f.ActiveProfile {
			marker = "*"
		}
		fmt.Fprintf(out, "%s %s\n", marker, n)
	}
	return nil
}

func runProfileUse(path, name string) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	f.ActiveProfile = name
	return config.Save(path, f)
}

func runProfileRm(path, name string) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	delete(f.Profiles, name)
	if f.ActiveProfile == name {
		f.ActiveProfile = ""
	}
	return config.Save(path, f)
}
