package numbers

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeNumbers struct {
	owned    []client.Number
	inv      []client.Number
	bought   string
	released string
}

func (f *fakeNumbers) List(_ context.Context, _ string) ([]client.Number, string, error) {
	return f.owned, "", nil
}
func (f *fakeNumbers) SearchInventory(_ context.Context, _ string) ([]client.Number, error) {
	return f.inv, nil
}
func (f *fakeNumbers) Buy(_ context.Context, n string) error     { f.bought = n; return nil }
func (f *fakeNumbers) Release(_ context.Context, n string) error { f.released = n; return nil }

func TestList_RendersOwned(t *testing.T) {
	f := &fakeNumbers{owned: []client.Number{
		{Number: "+14155551212", Country: "US", MonthlyRentalRate: "1.00"},
		{Number: "+919999999999", Country: "IN", MonthlyRentalRate: "0.80"},
	}}
	var out bytes.Buffer
	if err := runList(f, &out, "table", 50, false); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"+14155551212", "+919999999999", "US", "IN"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestSearch_RendersInventory(t *testing.T) {
	f := &fakeNumbers{inv: []client.Number{
		{Number: "+12025550100", Country: "US", SetupRate: "0", MonthlyRentalRate: "1.00"},
	}}
	var out bytes.Buffer
	if err := runSearch(f, &out, "table", "US"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "+12025550100") {
		t.Fatalf("missing number:\n%s", out.String())
	}
}

func TestBuy_CallsAPI(t *testing.T) {
	f := &fakeNumbers{}
	var out bytes.Buffer
	if err := runBuy(f, &out, "+14155551212"); err != nil {
		t.Fatal(err)
	}
	if f.bought != "+14155551212" {
		t.Fatalf("bought = %q", f.bought)
	}
	if !strings.Contains(out.String(), "+14155551212") {
		t.Fatalf("output missing number:\n%s", out.String())
	}
}

func TestRelease_CallsAPI(t *testing.T) {
	f := &fakeNumbers{}
	var out bytes.Buffer
	if err := runRelease(f, &out, "+14155551212"); err != nil {
		t.Fatal(err)
	}
	if f.released != "+14155551212" {
		t.Fatalf("released = %q", f.released)
	}
}
