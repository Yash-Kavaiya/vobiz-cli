// Package runtime resolves credentials and builds typed clients for command implementations.
package runtime

import (
	"os"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

type Overrides struct {
	ConfigPath  string
	Profile     string
	FlagID      string
	FlagToken   string
	FlagBaseURL string
}

func ResolveCreds(o Overrides) (cliAuth.Credentials, error) {
	path := o.ConfigPath
	if path == "" {
		var err error
		path, err = config.DefaultPath()
		if err != nil {
			return cliAuth.Credentials{}, err
		}
	}
	f, err := config.Load(path)
	if err != nil {
		return cliAuth.Credentials{}, err
	}
	return cliAuth.Resolve(cliAuth.Inputs{
		Config:      f,
		Profile:     o.Profile,
		FlagID:      o.FlagID,
		FlagToken:   o.FlagToken,
		FlagBaseURL: o.FlagBaseURL,
		EnvID:       os.Getenv("VOBIZ_AUTH_ID"),
		EnvToken:    os.Getenv("VOBIZ_AUTH_TOKEN"),
	})
}

func NewClient(o Overrides) (*client.Client, error) {
	creds, err := ResolveCreds(o)
	if err != nil {
		return nil, err
	}
	return client.New(creds), nil
}
