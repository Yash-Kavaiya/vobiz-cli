package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	in := &File{
		ActiveProfile: "default",
		Profiles: map[string]Profile{
			"default": {AuthID: "AB12", AuthToken: "tok", BaseURL: "https://api.vobiz.ai/api/v1"},
			"staging": {AuthID: "ZZ99", AuthToken: "stg", BaseURL: "https://api.vobiz.ai/api/v1"},
		},
	}

	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
		}
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.ActiveProfile != "default" {
		t.Fatalf("active = %q", out.ActiveProfile)
	}
	if out.Profiles["staging"].AuthID != "ZZ99" {
		t.Fatalf("staging auth id = %q", out.Profiles["staging"].AuthID)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.yaml")
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load missing: unexpected err %v", err)
	}
	if len(out.Profiles) != 0 || out.ActiveProfile != "" {
		t.Fatalf("expected empty file struct, got %+v", out)
	}
}

func TestDefaultPathUsesHomeDir(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	t.Setenv("USERPROFILE", `C:\fakehome`) // windows
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(got) != "config.yaml" {
		t.Fatalf("DefaultPath = %q", got)
	}
	if filepath.Base(filepath.Dir(got)) != ".vobiz" {
		t.Fatalf("DefaultPath parent = %q", filepath.Dir(got))
	}
}
