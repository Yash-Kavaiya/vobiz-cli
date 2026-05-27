package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestNumbers_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Number/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objects": []map[string]any{
				{"number": "+14155551212", "country": "US", "monthly_rental_rate": "1.00"},
			},
			"meta": map[string]any{"next": ""},
		})
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	items, next, err := c.Numbers.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Number != "+14155551212" {
		t.Fatalf("got %+v", items)
	}
	if next != "" {
		t.Fatalf("next = %q", next)
	}
}

func TestNumbers_SearchInventory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/Account/AB12/PhoneNumber/") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("country_iso") != "IN" {
			t.Errorf("country_iso = %q", r.URL.Query().Get("country_iso"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objects": []map[string]any{
				{"number": "+919999999999", "country": "IN", "setup_rate": "0", "monthly_rental_rate": "0.80"},
			},
		})
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Numbers.SearchInventory(context.Background(), "IN")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != "+919999999999" {
		t.Fatalf("got %+v", got)
	}
}

func TestNumbers_Buy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		if !strings.Contains(r.URL.Path, "AvailablePrefix") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Errorf("missing idempotency key")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"created","numbers":["+14155551212"]}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Numbers.Buy(context.Background(), "+14155551212"); err != nil {
		t.Fatal(err)
	}
}

func TestNumbers_Release(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/Account/AB12/Number/") {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Numbers.Release(context.Background(), "+14155551212"); err != nil {
		t.Fatal(err)
	}
}
