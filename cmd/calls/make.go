package calls

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type makeFlags struct {
	AnswerMethod         string
	RingURL              string
	HangupURL            string
	FallbackURL          string
	MachineDetection     string
	MachineDetectionTime int
	CallerName           string
	TimeLimit            int
}

func newMakeCmd() *cobra.Command {
	var (
		from, to, ans string
		flags         makeFlags
	)
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Place an outbound call",
		Long:  "Place an outbound call. The Vobiz API will request --answer-url when the callee picks up; that URL must return valid Vobiz XML.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := CallsFactory()
			if err != nil {
				return err
			}
			return runMake(a, cmd.OutOrStdout(), from, to, ans, flags)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "caller ID in E.164 (required)")
	cmd.Flags().StringVar(&to, "to", "", "destination number(s) in E.164, separated by '<' for fan-out (required)")
	cmd.Flags().StringVar(&ans, "answer-url", "", "URL returning Vobiz XML when the call connects (required)")
	cmd.Flags().StringVar(&flags.AnswerMethod, "answer-method", "", "HTTP verb for --answer-url (default POST)")
	cmd.Flags().StringVar(&flags.RingURL, "ring-url", "", "URL notified when the call starts ringing")
	cmd.Flags().StringVar(&flags.HangupURL, "hangup-url", "", "URL notified when the call hangs up")
	cmd.Flags().StringVar(&flags.FallbackURL, "fallback-url", "", "URL invoked if --answer-url fails")
	cmd.Flags().StringVar(&flags.MachineDetection, "machine-detection", "", "answering-machine detection: true|hangup")
	cmd.Flags().IntVar(&flags.MachineDetectionTime, "machine-detection-time", 0, "ms to wait for AMD (2000-10000)")
	cmd.Flags().StringVar(&flags.CallerName, "caller-name", "", "caller display name (max 50 chars)")
	cmd.Flags().IntVar(&flags.TimeLimit, "time-limit", 0, "max call duration in seconds (default 14400)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("answer-url")
	return cmd
}

func runMake(api client.CallsAPI, w io.Writer, from, to, ans string, f makeFlags) error {
	resp, err := api.Make(context.Background(), client.MakeCallParams{
		From: from, To: to, AnswerURL: ans,
		AnswerMethod:         f.AnswerMethod,
		RingURL:              f.RingURL,
		HangupURL:            f.HangupURL,
		FallbackURL:          f.FallbackURL,
		MachineDetection:     f.MachineDetection,
		MachineDetectionTime: f.MachineDetectionTime,
		CallerName:           f.CallerName,
		TimeLimit:            f.TimeLimit,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\nRequestUUID: %s\nAPIID:       %s\n", resp.Message, resp.RequestUUID, resp.APIID)
	return nil
}
