package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/telemetry/tracing"
)

var traceLimit int

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Inspect distributed execution traces and module spans (AIA-19 §8)",
}

var traceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent operational spans across autonomous subsystems",
	RunE: func(cmd *cobra.Command, args []string) error {
		tracer := tracing.DefaultTracer()
		spans := tracer.GetRecentSpans(traceLimit)
		if len(spans) == 0 {
			fmt.Println("No recorded spans in active session buffer.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "TRACE ID\tSPAN ID\tOPERATION\tSTATUS\tDURATION\tSTART TIME")
		fmt.Fprintln(w, "--------\t-------\t---------\t------\t--------\t----------")
		for _, s := range spans {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%dms\t%s\n",
				s.TraceID,
				s.SpanID[:8],
				s.Name,
				s.Status,
				s.DurationMs,
				s.StartTime.Format("15:04:05.000"),
			)
		}
		w.Flush()
		return nil
	},
}

var traceShowCmd = &cobra.Command{
	Use:   "show <trace-id>",
	Short: "Show all spans and attributes for a specific trace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tracer := tracing.DefaultTracer()
		spans := tracer.GetSpansByTrace(args[0])
		if len(spans) == 0 {
			fmt.Printf("No spans found for trace ID: %s\n", args[0])
			return nil
		}

		fmt.Printf("Trace ID: %s (%d spans)\n", args[0], len(spans))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SPAN ID\tPARENT ID\tOPERATION\tSTATUS\tDURATION\tATTRIBUTES")
		fmt.Fprintln(w, "-------\t---------\t---------\t------\t--------\t----------")
		for _, s := range spans {
			parent := s.ParentSpanID
			if len(parent) > 8 {
				parent = parent[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%dms\t%+v\n",
				s.SpanID[:8],
				parent,
				s.Name,
				s.Status,
				s.DurationMs,
				s.Attributes,
			)
		}
		w.Flush()
		return nil
	},
}

func init() {
	traceListCmd.Flags().IntVarP(&traceLimit, "limit", "l", 50, "Maximum spans to display")
	traceCmd.AddCommand(traceListCmd)
	traceCmd.AddCommand(traceShowCmd)

	RootCmd.AddCommand(traceCmd)
}
