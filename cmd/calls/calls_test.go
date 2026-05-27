package calls

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeCalls struct {
	madeWith client.MakeCallParams
	cdr      *client.CDR
	cdrs     []client.CDR
	pag      client.Pagination
}

func (f *fakeCalls) Make(_ context.Context, p client.MakeCallParams) (*client.MakeCallResponse, error) {
	f.madeWith = p
	return &client.MakeCallResponse{APIID: "a1", Message: "submitted", RequestUUID: "uuid-1"}, nil
}
func (f *fakeCalls) ListCDR(_ context.Context, _ client.CDRListOpts) ([]client.CDR, client.Pagination, error) {
	return f.cdrs, f.pag, nil
}
func (f *fakeCalls) GetCDR(_ context.Context, _ string) (*client.CDR, error) {
	return f.cdr, nil
}

func TestMake_PassesParams_AndPrintsRequestUUID(t *testing.T) {
	f := &fakeCalls{}
	var out bytes.Buffer
	if err := runMake(f, &out, "+14150000000", "+14155551212", "https://x/ans", makeFlags{}); err != nil {
		t.Fatal(err)
	}
	if f.madeWith.From != "+14150000000" || f.madeWith.To != "+14155551212" || f.madeWith.AnswerURL != "https://x/ans" {
		t.Fatalf("params: %+v", f.madeWith)
	}
	if !strings.Contains(out.String(), "uuid-1") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestList_TableOutput(t *testing.T) {
	f := &fakeCalls{
		cdrs: []client.CDR{
			{UUID: "c1", CallerIDNumber: "+1", DestinationNumber: "+2", Duration: 30, BillSec: 25, Cost: "0.10", HangupCause: "NORMAL_CLEARING", CallDirection: "outbound"},
		},
		pag: client.Pagination{Page: 1, PerPage: 20, Total: 1, Pages: 1},
	}
	var out bytes.Buffer
	if err := runList(f, &out, "table", listFlags{Page: 1, PerPage: 20}); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"c1", "+1", "+2", "30", "NORMAL_CLEARING"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestGet_Prints(t *testing.T) {
	f := &fakeCalls{cdr: &client.CDR{UUID: "c1", Duration: 42, Cost: "0.05"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "table", "c1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "c1") {
		t.Fatalf("missing uuid:\n%s", out.String())
	}
}
