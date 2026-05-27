package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Number struct {
	Number            string `json:"number"               yaml:"number"`
	Country           string `json:"country"              yaml:"country"`
	NumberType        string `json:"number_type,omitempty" yaml:"number_type,omitempty"`
	MonthlyRentalRate string `json:"monthly_rental_rate"  yaml:"monthly_rental_rate"`
	SetupRate         string `json:"setup_rate,omitempty" yaml:"setup_rate,omitempty"`
	Application       string `json:"application,omitempty" yaml:"application,omitempty"`
	AddedOn           string `json:"added_on,omitempty"    yaml:"added_on,omitempty"`
}

type NumbersAPI interface {
	List(ctx context.Context, cursor string) ([]Number, string, error)
	SearchInventory(ctx context.Context, countryISO string) ([]Number, error)
	Buy(ctx context.Context, number string) error
	Release(ctx context.Context, number string) error
}

type numbersAPI struct {
	http   *httpx.Client
	authID string
}

func (n *numbersAPI) List(ctx context.Context, cursor string) ([]Number, string, error) {
	path := "/Account/" + n.authID + "/Number/"
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	var raw struct {
		Objects []Number `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := n.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (n *numbersAPI) SearchInventory(ctx context.Context, countryISO string) ([]Number, error) {
	q := url.Values{}
	if countryISO != "" {
		q.Set("country_iso", countryISO)
	}
	path := "/Account/" + n.authID + "/PhoneNumber/"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var raw struct {
		Objects []Number `json:"objects"`
	}
	if err := n.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Objects, nil
}

func (n *numbersAPI) Buy(ctx context.Context, number string) error {
	path := "/Account/" + n.authID + "/AvailablePrefix/" + url.PathEscape(number) + "/"
	return n.http.DoJSON(ctx, http.MethodPost, path, struct{}{}, nil)
}

func (n *numbersAPI) Release(ctx context.Context, number string) error {
	path := "/Account/" + n.authID + "/Number/" + url.PathEscape(number) + "/"
	return n.http.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}
