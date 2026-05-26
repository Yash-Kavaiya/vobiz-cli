package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func TestSmoke_AccountGetEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-ID") != "AB12" || r.Header.Get("X-Auth-Token") != "tok" {
			t.Errorf("auth headers missing: %+v", r.Header)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"auth_id":      "AB12",
			"account_type": "developer",
			"billing_mode": "prepaid",
			"timezone":     "UTC",
			"cash_credits": "100.00",
		})
	}))
	defer srv.Close()

	// Point the CLI at a temp HOME so config.DefaultPath() finds our test file.
	// os.UserHomeDir() honors HOME on POSIX and USERPROFILE on Windows.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	if err := config.Save(filepath.Join(dir, ".vobiz", "config.yaml"), &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL},
		},
	}); err != nil {
		t.Fatal(err)
	}

	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"account", "get", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"auth_id": "AB12"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
}
