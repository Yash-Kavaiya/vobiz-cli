package auth

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

type fakeAccountAPI struct {
	getErr error
	called bool
}

func (f *fakeAccountAPI) Get(_ context.Context) (*client.Account, error) {
	f.called = true
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &client.Account{AuthID: "AB12", AccountType: "developer"}, nil
}
func (f *fakeAccountAPI) Balance(_ context.Context) (string, error) { return "", nil }
func (f *fakeAccountAPI) Transactions(_ context.Context, _ string, _ int) ([]client.Transaction, string, error) {
	return nil, "", nil
}
func (f *fakeAccountAPI) Concurrency(_ context.Context) (*client.Concurrency, error) {
	return nil, nil
}

func TestRunLogin_WritesConfigAndVerifies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	fake := &fakeAccountAPI{}
	var out bytes.Buffer
	err := runLogin(loginInputs{
		ConfigPath: path,
		Profile:    "default",
		AuthID:     "AB12",
		AuthToken:  "tok",
		BaseURL:    "https://api.vobiz.ai/api/v1",
		Out:        &out,
		VerifyAcct: func(_ string) accountVerifier { return fake },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !fake.called {
		t.Fatal("verifier not called")
	}
	if !strings.Contains(out.String(), "saved") {
		t.Fatalf("output: %q", out.String())
	}
	f, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Profiles["default"].AuthID != "AB12" {
		t.Fatalf("config: %+v", f)
	}
	if f.ActiveProfile != "default" {
		t.Fatalf("active profile = %q", f.ActiveProfile)
	}
}

func TestRunLogin_VerificationFailureDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	fake := &fakeAccountAPI{getErr: errors.New("401 unauthorized")}
	var out bytes.Buffer
	err := runLogin(loginInputs{
		ConfigPath: path,
		Profile:    "default",
		AuthID:     "AB12",
		AuthToken:  "bad",
		Out:        &out,
		VerifyAcct: func(_ string) accountVerifier { return fake },
	})
	if err == nil {
		t.Fatal("expected error")
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["default"]; ok {
		t.Fatal("profile should not have been written on verification failure")
	}
}
