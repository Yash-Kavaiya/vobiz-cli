package client

import (
	"context"
	"net/http"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Account struct {
	AuthID       string `json:"auth_id"       yaml:"auth_id"`
	AccountType  string `json:"account_type"  yaml:"account_type"`
	BillingMode  string `json:"billing_mode"  yaml:"billing_mode"`
	Timezone     string `json:"timezone"      yaml:"timezone"`
	CashCredits  string `json:"cash_credits"  yaml:"cash_credits"`
	AutoRecharge bool   `json:"auto_recharge" yaml:"auto_recharge"`
	ResourceURI  string `json:"resource_uri"  yaml:"resource_uri"`
}

type Transaction struct {
	ID          string `json:"id"          yaml:"id"`
	Amount      string `json:"amount"      yaml:"amount"`
	Description string `json:"description" yaml:"description"`
	CreatedAt   string `json:"created_at"  yaml:"created_at"`
}

type Concurrency struct {
	Limit   int `json:"limit"   yaml:"limit"`
	Current int `json:"current" yaml:"current"`
}

type AccountAPI interface {
	Get(ctx context.Context) (*Account, error)
	Balance(ctx context.Context) (string, error)
	Transactions(ctx context.Context, cursor string, limit int) ([]Transaction, string, error)
	Concurrency(ctx context.Context) (*Concurrency, error)
}

type accountAPI struct {
	http   *httpx.Client
	authID string
}

func (a *accountAPI) Get(ctx context.Context) (*Account, error) {
	var out Account
	if err := a.http.DoJSON(ctx, http.MethodGet, "/Account/"+a.authID+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *accountAPI) Balance(ctx context.Context) (string, error) {
	acc, err := a.Get(ctx)
	if err != nil {
		return "", err
	}
	return acc.CashCredits, nil
}

func (a *accountAPI) Transactions(ctx context.Context, cursor string, limit int) ([]Transaction, string, error) {
	path := "/Account/" + a.authID + "/Transaction/"
	if cursor != "" {
		path += "?cursor=" + cursor
	}
	var raw struct {
		Objects []Transaction `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := a.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (a *accountAPI) Concurrency(ctx context.Context) (*Concurrency, error) {
	var out Concurrency
	if err := a.http.DoJSON(ctx, http.MethodGet, "/Account/"+a.authID+"/Concurrency/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
