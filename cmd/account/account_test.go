package account

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeAccount struct {
	acc         *client.Account
	txs         []client.Transaction
	next        string
	concurrency *client.Concurrency
	wasCalled   string
}

func (f *fakeAccount) Get(_ context.Context) (*client.Account, error) {
	f.wasCalled = "get"
	return f.acc, nil
}
func (f *fakeAccount) Balance(_ context.Context) (string, error) {
	f.wasCalled = "balance"
	return f.acc.CashCredits, nil
}
func (f *fakeAccount) Transactions(_ context.Context, _ string, _ int) ([]client.Transaction, string, error) {
	f.wasCalled = "transactions"
	return f.txs, f.next, nil
}
func (f *fakeAccount) Concurrency(_ context.Context) (*client.Concurrency, error) {
	f.wasCalled = "concurrency"
	return f.concurrency, nil
}

func TestGet_TableOutput(t *testing.T) {
	f := &fakeAccount{acc: &client.Account{AuthID: "AB12", AccountType: "developer", BillingMode: "prepaid", CashCredits: "25.00", Timezone: "Asia/Kolkata"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "table"); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"AB12", "developer", "prepaid", "25.00"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q in:\n%s", w, out.String())
		}
	}
}

func TestGet_JSONOutput(t *testing.T) {
	f := &fakeAccount{acc: &client.Account{AuthID: "AB12", AccountType: "developer", CashCredits: "1.50"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "json"); err != nil {
		t.Fatal(err)
	}
	var got client.Account
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not valid json: %v\n%s", err, out.String())
	}
	if got.AuthID != "AB12" || got.CashCredits != "1.50" {
		t.Fatalf("decoded = %+v", got)
	}
}

func TestBalance_Prints(t *testing.T) {
	f := &fakeAccount{acc: &client.Account{CashCredits: "12.34"}}
	var out bytes.Buffer
	if err := runBalance(f, &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "12.34" {
		t.Fatalf("balance: %q", out.String())
	}
}

func TestTransactions_Pages(t *testing.T) {
	f := &fakeAccount{txs: []client.Transaction{
		{ID: "1", Amount: "10", Description: "topup", CreatedAt: "2026-05-23"},
		{ID: "2", Amount: "-1", Description: "call", CreatedAt: "2026-05-23"},
	}}
	var out bytes.Buffer
	if err := runTransactions(f, &out, "table", 10, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "topup") {
		t.Fatalf("missing topup:\n%s", out.String())
	}
}

func TestConcurrency_Prints(t *testing.T) {
	f := &fakeAccount{concurrency: &client.Concurrency{Limit: 50, Current: 3}}
	var out bytes.Buffer
	if err := runConcurrency(f, &out, "table"); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"50", "3"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}
