package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestCalls_Make(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Call/" || r.Method != http.MethodPost {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["from"] != "+14150000000" || req["to"] != "+14155551212" {
			t.Errorf("from/to wrong: %+v", req)
		}
		if req["answer_url"] != "https://example.com/ans" {
			t.Errorf("answer_url wrong: %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"api_id":"a1","message":"call submitted","request_uuid":"uuid-1"}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Calls.Make(context.Background(), MakeCallParams{
		From: "+14150000000", To: "+14155551212", AnswerURL: "https://example.com/ans",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestUUID != "uuid-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestCalls_ListCDR_Pagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cdr") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("page = %q", r.URL.Query().Get("page"))
		}
		_, _ = w.Write([]byte(`{
		  "data":[{"uuid":"c1","caller_id_number":"+1","destination_number":"+2","duration":30,"billsec":25,"cost":"0.10","call_direction":"outbound","hangup_cause":"NORMAL_CLEARING"}],
		  "pagination":{"page":2,"per_page":20,"total":50,"pages":3,"has_next":true,"has_prev":true},
		  "success":true
		}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	rows, page, err := c.Calls.ListCDR(context.Background(), CDRListOpts{Page: 2, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].UUID != "c1" {
		t.Fatalf("rows: %+v", rows)
	}
	if !page.HasNext || page.Total != 50 {
		t.Fatalf("page: %+v", page)
	}
}

func TestCalls_GetCDR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cdr/c1") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"uuid":"c1","caller_id_number":"+1","destination_number":"+2","duration":30,"cost":"0.10"}}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Calls.GetCDR(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "c1" {
		t.Fatalf("got %+v", got)
	}
}
