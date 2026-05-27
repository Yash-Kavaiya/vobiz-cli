package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type MakeCallParams struct {
	From                 string `json:"from"`
	To                   string `json:"to"`
	AnswerURL            string `json:"answer_url"`
	AnswerMethod         string `json:"answer_method,omitempty"`
	RingURL              string `json:"ring_url,omitempty"`
	RingMethod           string `json:"ring_method,omitempty"`
	HangupURL            string `json:"hangup_url,omitempty"`
	HangupMethod         string `json:"hangup_method,omitempty"`
	FallbackURL          string `json:"fallback_url,omitempty"`
	FallbackMethod       string `json:"fallback_method,omitempty"`
	MachineDetection     string `json:"machine_detection,omitempty"`
	MachineDetectionTime int    `json:"machine_detection_time,omitempty"`
	CallerName           string `json:"caller_name,omitempty"`
	SendDigits           string `json:"send_digits,omitempty"`
	TimeLimit            int    `json:"time_limit,omitempty"`
}

type MakeCallResponse struct {
	APIID       string `json:"api_id"        yaml:"api_id"`
	Message     string `json:"message"       yaml:"message"`
	RequestUUID string `json:"request_uuid"  yaml:"request_uuid"`
}

// CDR is a Call Detail Record. The Vobiz API returns 40+ fields per record;
// these are the most useful for terminal display. Callers who need fields not
// modeled here can use -o json to read the raw payload.
type CDR struct {
	UUID              string `json:"uuid"               yaml:"uuid"`
	CallerIDNumber    string `json:"caller_id_number"   yaml:"caller_id_number"`
	DestinationNumber string `json:"destination_number" yaml:"destination_number"`
	CallDirection     string `json:"call_direction"     yaml:"call_direction"`
	Duration          int    `json:"duration"           yaml:"duration"`
	BillSec           int    `json:"billsec"            yaml:"billsec"`
	Cost              string `json:"cost"               yaml:"cost"`
	HangupCause       string `json:"hangup_cause"       yaml:"hangup_cause"`
	StartTime         string `json:"start_stamp,omitempty"  yaml:"start_stamp,omitempty"`
	EndTime           string `json:"end_stamp,omitempty"    yaml:"end_stamp,omitempty"`
	MOS               string `json:"mos,omitempty"          yaml:"mos,omitempty"`
}

type Pagination struct {
	Page    int  `json:"page"      yaml:"page"`
	PerPage int  `json:"per_page"  yaml:"per_page"`
	Total   int  `json:"total"     yaml:"total"`
	Pages   int  `json:"pages"     yaml:"pages"`
	HasNext bool `json:"has_next"  yaml:"has_next"`
	HasPrev bool `json:"has_prev"  yaml:"has_prev"`
}

type CDRListOpts struct {
	Page          int
	PerPage       int
	FromNumber    string
	ToNumber      string
	StartDate     string // ISO date
	EndDate       string
	CallDirection string // "inbound" | "outbound"
	MinDuration   int    // seconds
}

type CallsAPI interface {
	Make(ctx context.Context, p MakeCallParams) (*MakeCallResponse, error)
	ListCDR(ctx context.Context, opts CDRListOpts) ([]CDR, Pagination, error)
	GetCDR(ctx context.Context, callID string) (*CDR, error)
}

type callsAPI struct {
	http   *httpx.Client
	authID string
}

func (c *callsAPI) Make(ctx context.Context, p MakeCallParams) (*MakeCallResponse, error) {
	var out MakeCallResponse
	if err := c.http.DoJSON(ctx, http.MethodPost, "/Account/"+c.authID+"/Call/", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *callsAPI) ListCDR(ctx context.Context, opts CDRListOpts) ([]CDR, Pagination, error) {
	q := url.Values{}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if opts.FromNumber != "" {
		q.Set("from_number", opts.FromNumber)
	}
	if opts.ToNumber != "" {
		q.Set("to_number", opts.ToNumber)
	}
	if opts.StartDate != "" {
		q.Set("start_date", opts.StartDate)
	}
	if opts.EndDate != "" {
		q.Set("end_date", opts.EndDate)
	}
	if opts.CallDirection != "" {
		q.Set("call_direction", opts.CallDirection)
	}
	if opts.MinDuration > 0 {
		q.Set("min_duration", strconv.Itoa(opts.MinDuration))
	}
	path := "/Account/" + c.authID + "/cdr"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var raw struct {
		Data       []CDR      `json:"data"`
		Pagination Pagination `json:"pagination"`
	}
	if err := c.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, Pagination{}, err
	}
	return raw.Data, raw.Pagination, nil
}

func (c *callsAPI) GetCDR(ctx context.Context, callID string) (*CDR, error) {
	var raw struct {
		Data CDR `json:"data"`
	}
	path := "/Account/" + c.authID + "/cdr/" + url.PathEscape(callID)
	if err := c.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return &raw.Data, nil
}
