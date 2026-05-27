package client

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Recording struct {
	RecordingID     string `json:"recording_id"     yaml:"recording_id"`
	CallUUID        string `json:"call_uuid"        yaml:"call_uuid"`
	Duration        int    `json:"duration"         yaml:"duration"`
	RecordingFormat string `json:"recording_format" yaml:"recording_format"`
	RecordingURL    string `json:"recording_url,omitempty" yaml:"recording_url,omitempty"`
	ResourceURI     string `json:"resource_uri"     yaml:"resource_uri"`
	AddedOn         string `json:"added_on,omitempty" yaml:"added_on,omitempty"`
}

type RecordingsAPI interface {
	List(ctx context.Context, cursor string) ([]Recording, string, error)
	Get(ctx context.Context, recordingID string) (*Recording, error)
	Download(ctx context.Context, recordingID string, dst io.Writer) error
}

type recordingsAPI struct {
	http   *httpx.Client
	authID string
}

func (r *recordingsAPI) List(ctx context.Context, cursor string) ([]Recording, string, error) {
	path := "/Account/" + r.authID + "/Recording/"
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	var raw struct {
		Objects []Recording `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := r.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (r *recordingsAPI) Get(ctx context.Context, recordingID string) (*Recording, error) {
	var out Recording
	path := "/Account/" + r.authID + "/Recording/" + url.PathEscape(recordingID)
	if err := r.http.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *recordingsAPI) Download(ctx context.Context, recordingID string, dst io.Writer) error {
	path := "/Account/" + r.authID + "/Recording/" + url.PathEscape(recordingID) + "/download"
	resp, err := r.http.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(dst, resp.Body)
	return err
}
