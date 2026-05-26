// Package auth resolves Vobiz credentials from flags, env vars, and config.
package auth

import (
	"fmt"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
	cliErrors "github.com/yash-kavaiya/vobiz-cli/internal/errors"
)

const DefaultBaseURL = "https://api.vobiz.ai/api/v1"

type Credentials struct {
	AuthID    string
	AuthToken string
	BaseURL   string
	Source    string // "flag" | "env" | "profile:<name>"
}

type Inputs struct {
	Config            *config.File
	Profile           string
	FlagID, FlagToken string
	EnvID, EnvToken   string
	FlagBaseURL       string
}

func Resolve(in Inputs) (Credentials, error) {
	c := Credentials{BaseURL: DefaultBaseURL}

	switch {
	case in.FlagID != "" && in.FlagToken != "":
		c.AuthID, c.AuthToken, c.Source = in.FlagID, in.FlagToken, "flag"
	case in.EnvID != "" && in.EnvToken != "":
		c.AuthID, c.AuthToken, c.Source = in.EnvID, in.EnvToken, "env"
	default:
		if in.Config == nil {
			return c, fmt.Errorf("%w: no credentials supplied (run 'vobiz auth login')", cliErrors.ErrAuth)
		}
		name := in.Profile
		if name == "" {
			name = in.Config.ActiveProfile
		}
		if name == "" {
			name = "default"
		}
		p, ok := in.Config.Profiles[name]
		if !ok {
			return c, fmt.Errorf("%w: profile %q not found (run 'vobiz auth login --profile %s')", cliErrors.ErrAuth, name, name)
		}
		if p.AuthID == "" || p.AuthToken == "" {
			return c, fmt.Errorf("%w: profile %q is missing auth_id or auth_token", cliErrors.ErrAuth, name)
		}
		c.AuthID, c.AuthToken, c.Source = p.AuthID, p.AuthToken, "profile:"+name
		if p.BaseURL != "" {
			c.BaseURL = p.BaseURL
		}
	}

	if in.FlagBaseURL != "" {
		c.BaseURL = in.FlagBaseURL
	}
	return c, nil
}
