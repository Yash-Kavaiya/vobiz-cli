package auth

import (
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func TestResolve_FlagBeatsEnvBeatsProfile(t *testing.T) {
	cfg := &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "PROF_ID", AuthToken: "PROF_TOK", BaseURL: "https://api.vobiz.ai/api/v1"},
		},
	}

	t.Run("profile when nothing overrides", func(t *testing.T) {
		got, err := Resolve(Inputs{Config: cfg})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "PROF_ID" || got.AuthToken != "PROF_TOK" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("env beats profile", func(t *testing.T) {
		got, err := Resolve(Inputs{
			Config: cfg,
			EnvID:  "ENV_ID", EnvToken: "ENV_TOK",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "ENV_ID" || got.AuthToken != "ENV_TOK" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("flag beats env", func(t *testing.T) {
		got, err := Resolve(Inputs{
			Config: cfg,
			EnvID:  "ENV_ID", EnvToken: "ENV_TOK",
			FlagID: "FLAG_ID", FlagToken: "FLAG_TOK",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "FLAG_ID" || got.AuthToken != "FLAG_TOK" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("named profile override", func(t *testing.T) {
		cfg.Profiles["staging"] = config.Profile{AuthID: "STG_ID", AuthToken: "STG_TOK"}
		got, err := Resolve(Inputs{Config: cfg, Profile: "staging"})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "STG_ID" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("missing returns ErrAuth", func(t *testing.T) {
		_, err := Resolve(Inputs{Config: &config.File{Profiles: map[string]config.Profile{}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolve_DefaultsBaseURL(t *testing.T) {
	cfg := &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "X", AuthToken: "Y"},
		},
	}
	got, _ := Resolve(Inputs{Config: cfg})
	if got.BaseURL != "https://api.vobiz.ai/api/v1" {
		t.Fatalf("BaseURL = %q", got.BaseURL)
	}
}
