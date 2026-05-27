// Package client exposes typed resource APIs over the shared httpx client.
package client

import (
	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
	"github.com/yash-kavaiya/vobiz-cli/internal/version"
)

type Client struct {
	HTTP    *httpx.Client
	Account AccountAPI
	Numbers NumbersAPI
	Calls   CallsAPI
}

func New(creds auth.Credentials) *Client {
	h := httpx.New(httpx.Config{
		BaseURL:   creds.BaseURL,
		AuthID:    creds.AuthID,
		AuthToken: creds.AuthToken,
		UserAgent: "vobiz-cli/" + version.Version,
	})
	return &Client{
		HTTP:    h,
		Account: &accountAPI{http: h, authID: creds.AuthID},
		Numbers: &numbersAPI{http: h, authID: creds.AuthID},
		Calls:   &callsAPI{http: h, authID: creds.AuthID},
	}
}
