package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var executionCmd = &cobra.Command{
	Use:   "execution",
	Short: "Inspect execution details and event stream",
}

var executionShowCmd = &cobra.Command{
	Use:   "show <execution-id>",
	Short: "Show details for an execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		e, err := appInst.ExecutionSvc.Get(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Println("=== Execution Details ===")
		fmt.Printf("ID:              %s\n", e.ID)
		fmt.Printf("Task ID:         %s\n", e.TaskID)
		fmt.Printf("Agent ID:        %s\n", e.AgentID)
		fmt.Printf("Provider:        %s\n", e.Provider)
		fmt.Printf("Status:          %s\n", e.Status)
		fmt.Printf("Conversation ID: %s\n", e.ConversationID)
		fmt.Printf("Workspace:       %s\n", e.Workspace)
		fmt.Printf("Created At:      %s\n", e.CreatedAt.Format("2006-01-02 15:04:05"))
		if e.StartedAt != nil {
			fmt.Printf("Started At:      %s\n", e.StartedAt.Format("2006-01-02 15:04:05"))
		}
		if e.EndedAt != nil {
			fmt.Printf("Ended At:        %s\n", e.EndedAt.Format("2006-01-02 15:04:05"))
		}
		if e.ErrorMessage != "" {
			fmt.Printf("Error:           %s\n", e.ErrorMessage)
		}
		return nil
	},
}

var executionEventsCmd = &cobra.Command{
	Use:   "events <execution-id>",
	Short: "List streamed events for an execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		events, err := appInst.ExecutionSvc.ListEvents(ctx, args[0])
		if err != nil {
			return err
		}

		if len(events) == 0 {
			fmt.Println("No events found for this execution.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SEQ\tTYPE\tSOURCE\tPAYLOAD")
		for _, ev := range events {
			payloadPreview := string(ev.Payload)
			if len(payloadPreview) > 60 {
				payloadPreview = payloadPreview[:60] + "..."
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", ev.Sequence, ev.EventType, ev.Source, payloadPreview)
		}
		_ = w.Flush()
		return nil
	},
}

func init() {
	executionCmd.AddCommand(executionShowCmd)
	executionCmd.AddCommand(executionEventsCmd)
}
