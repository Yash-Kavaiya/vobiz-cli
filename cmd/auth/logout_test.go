package auth

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func writeCfg(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	f := &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "A", AuthToken: "T"},
			"staging": {AuthID: "B", AuthToken: "U"},
		},
	}
	if err := config.Save(path, f); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunLogout_RemovesActiveProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)

	var out bytes.Buffer
	if err := runLogout(path, "", &out); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["default"]; ok {
		t.Fatal("default profile should be removed")
	}
	if f.ActiveProfile == "default" {
		t.Fatal("active profile should be cleared or rotated")
	}
}

func TestRunLogout_NamedProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)

	if err := runLogout(path, "staging", new(bytes.Buffer)); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["staging"]; ok {
		t.Fatal("staging profile should be removed")
	}
	if f.ActiveProfile != "default" {
		t.Fatalf("active = %q", f.ActiveProfile)
	}
}

func TestRunStatus_PrintsActiveProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)

	var out bytes.Buffer
	if err := runStatus(path, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("default")) {
		t.Fatalf("status: %q", out.String())
	}
}

func TestRunProfileList(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)
	var out bytes.Buffer
	if err := runProfileList(path, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"default", "staging"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("profile list missing %q: %s", want, out.String())
		}
	}
}

func TestRunProfileUse(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)
	if err := runProfileUse(path, "staging"); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if f.ActiveProfile != "staging" {
		t.Fatalf("active = %q", f.ActiveProfile)
	}
}

func TestRunProfileRm(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)
	if err := runProfileRm(path, "staging"); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["staging"]; ok {
		t.Fatal("staging should be removed")
	}
}
