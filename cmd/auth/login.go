package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

type accountVerifier interface {
	Get(ctx context.Context) (*client.Account, error)
}

type loginInputs struct {
	ConfigPath string
	Profile    string
	AuthID     string
	AuthToken  string
	BaseURL    string
	Out        io.Writer
	VerifyAcct func(authID string) accountVerifier
}

func newLoginCmd() *cobra.Command {
	var (
		profile string
		baseURL string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save Vobiz Auth ID + Token to ~/.vobiz/config.yaml",
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, tok, err := promptCredentials(cmd.InOrStdin(), cmd.OutOrStdout())
			if err != nil {
				return err
			}
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runLogin(loginInputs{
				ConfigPath: path,
				Profile:    profile,
				AuthID:     id,
				AuthToken:  tok,
				BaseURL:    baseURL,
				Out:        cmd.OutOrStdout(),
				VerifyAcct: func(authID string) accountVerifier {
					c := client.New(cliAuth.Credentials{AuthID: authID, AuthToken: tok, BaseURL: pickBaseURL(baseURL)})
					return c.Account
				},
			})
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "default", "profile name to save under")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "override base URL")
	return cmd
}

func promptCredentials(in io.Reader, out io.Writer) (string, string, error) {
	fmt.Fprint(out, "Auth ID: ")
	r := bufio.NewReader(in)
	idLine, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}
	id := strings.TrimSpace(idLine)
	if id == "" {
		return "", "", errors.New("Auth ID is required")
	}

	fmt.Fprint(out, "Auth Token: ")
	var tok string
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", "", err
		}
		tok = strings.TrimSpace(string(b))
	} else {
		line, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", "", err
		}
		tok = strings.TrimSpace(line)
	}
	if tok == "" {
		return "", "", errors.New("Auth Token is required")
	}
	return id, tok, nil
}

func pickBaseURL(flag string) string {
	if flag != "" {
		return flag
	}
	return cliAuth.DefaultBaseURL
}

func runLogin(in loginInputs) error {
	verifier := in.VerifyAcct(in.AuthID)
	if _, err := verifier.Get(context.Background()); err != nil {
		return fmt.Errorf("credentials rejected by API: %w", err)
	}

	f, err := config.Load(in.ConfigPath)
	if err != nil {
		return err
	}
	if f.Profiles == nil {
		f.Profiles = map[string]config.Profile{}
	}
	name := in.Profile
	if name == "" {
		name = "default"
	}
	f.Profiles[name] = config.Profile{
		AuthID:    in.AuthID,
		AuthToken: in.AuthToken,
		BaseURL:   in.BaseURL,
	}
	if f.ActiveProfile == "" {
		f.ActiveProfile = name
	}
	if err := config.Save(in.ConfigPath, f); err != nil {
		return err
	}
	fmt.Fprintf(in.Out, "Credentials saved to %s (profile %q).\n", in.ConfigPath, name)
	return nil
}
