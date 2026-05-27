package client

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestRecordings_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Recording/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
		  "objects":[{"recording_id":"r1","call_uuid":"c1","duration":12,"recording_format":"mp3","resource_uri":"/Recording/r1"}],
		  "meta":{"next":""}
		}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	rows, _, err := c.Recordings.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RecordingID != "r1" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestRecordings_Download_StreamsBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/Recording/r1/download") {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("\x00ID3FAKE_MP3_BYTES"))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	var buf bytes.Buffer
	if err := c.Recordings.Download(context.Background(), "r1", &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("FAKE_MP3_BYTES")) {
		t.Fatalf("download body: %q", buf.String())
	}
}
