package calls

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
	"github.com/yash-kavaiya/vobiz-cli/internal/paginate"
)

var RecordingsFactory = func() (client.RecordingsAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Recordings, nil
}

func newRecordingsCmd(format func() string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recordings",
		Short: "List and download call recordings",
	}
	cmd.AddCommand(newRecordingsListCmd(format))
	cmd.AddCommand(newRecordingsDownloadCmd())
	return cmd
}

func newRecordingsListCmd(format func() string) *cobra.Command {
	var (
		limit int
		all   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recordings (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := RecordingsFactory()
			if err != nil {
				return err
			}
			return runRecordingsList(a, cmd.OutOrStdout(), format(), limit, all)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max number of rows")
	cmd.Flags().BoolVar(&all, "all", false, "fetch all pages")
	return cmd
}

func runRecordingsList(api client.RecordingsAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Recording], error) {
		items, next, err := api.List(ctx, cursor)
		if err != nil {
			return paginate.Page[client.Recording]{}, err
		}
		return paginate.Page[client.Recording]{Items: items, NextCursor: next}, nil
	}
	var (
		rows []client.Recording
		err  error
	)
	if all {
		rows, err = paginate.All(context.Background(), fetch)
	} else {
		rows, err = paginate.AllN(context.Background(), fetch, limit)
	}
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "ID", Field: "RecordingID"},
		{Header: "CALL UUID", Field: "CallUUID"},
		{Header: "DURATION", Field: "Duration"},
		{Header: "FORMAT", Field: "RecordingFormat"},
		{Header: "ADDED ON", Field: "AddedOn"},
	}
	return output.Render(w, rows, cols, f)
}

func newRecordingsDownloadCmd() *cobra.Command {
	var dest string
	cmd := &cobra.Command{
		Use:   "download <recording-id>",
		Short: "Download a recording's audio file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := RecordingsFactory()
			if err != nil {
				return err
			}
			out := dest
			if out == "" {
				out = args[0] + ".mp3"
			}
			return runRecordingsDownload(a, args[0], out)
		},
	}
	cmd.Flags().StringVarP(&dest, "output-file", "f", "", "destination path (default <recording-id>.mp3)")
	return cmd
}

func runRecordingsDownload(api client.RecordingsAPI, recordingID, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := api.Download(context.Background(), recordingID, f); err != nil {
		_ = os.Remove(dest)
		return err
	}
	fmt.Fprintf(os.Stderr, "Wrote %s.\n", dest)
	return nil
}
