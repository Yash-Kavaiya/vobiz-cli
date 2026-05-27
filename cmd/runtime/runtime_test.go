package runtime

import (
	"path/filepath"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func TestResolveCreds_PrefersEnvOverConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := config.Save(path, &config.File{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {AuthID: "FILE_ID", AuthToken: "FILE_TOK"}},
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VOBIZ_AUTH_ID", "ENV_ID")
	t.Setenv("VOBIZ_AUTH_TOKEN", "ENV_TOK")

	got, err := ResolveCreds(Overrides{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthID != "ENV_ID" || got.AuthToken != "ENV_TOK" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveCreds_FlagsBeatEverything(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	_ = config.Save(path, &config.File{Profiles: map[string]config.Profile{}})

	t.Setenv("VOBIZ_AUTH_ID", "ENV_ID")
	t.Setenv("VOBIZ_AUTH_TOKEN", "ENV_TOK")

	got, err := ResolveCreds(Overrides{
		ConfigPath: path,
		FlagID:     "FLAG_ID",
		FlagToken:  "FLAG_TOK",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthID != "FLAG_ID" {
		t.Fatalf("got %+v", got)
	}
}
