package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestAccount_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"auth_id":      "AB12",
			"account_type": "developer",
			"billing_mode": "prepaid",
			"timezone":     "Asia/Kolkata",
			"cash_credits": "25.00",
		})
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Account.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthID != "AB12" || got.AccountType != "developer" || got.CashCredits != "25.00" {
		t.Fatalf("%+v", got)
	}
}
