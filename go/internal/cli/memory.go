package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage long-term memory and memory candidates",
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active memories",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		memories, err := appInst.MemorySvc.Search(ctx, repository.MemoryQuery{Limit: 50})
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tSCOPE\tCONFIDENCE\tTITLE")
		for _, m := range memories {
			idShort := m.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%s\n", idShort, m.Type, m.Scope, m.Confidence, m.Title)
		}
		_ = w.Flush()
		return nil
	},
}

var memorySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search memories by keyword",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		memories, err := appInst.MemorySvc.Search(ctx, repository.MemoryQuery{
			Keyword: args[0],
			Limit:   20,
		})
		if err != nil {
			return err
		}

		if len(memories) == 0 {
			fmt.Println("No matching memories found.")
			return nil
		}

		for i, m := range memories {
			fmt.Printf("%d. [%s] %s (Confidence: %.2f)\n   %s\n\n", i+1, m.Type, m.Title, m.Confidence, m.Content)
		}
		return nil
	},
}

var candidateCmd = &cobra.Command{
	Use:   "candidate",
	Short: "Manage memory candidates",
}

var candidateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending memory candidates",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		candidates, err := appInst.MemorySvc.ListCandidates(ctx, memory.ReviewPending)
		if err != nil {
			return err
		}

		if len(candidates) == 0 {
			fmt.Println("No pending memory candidates.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tSCOPE\tTITLE\tCREATED AT")
		for _, c := range candidates {
			idShort := c.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", idShort, c.Type, c.Scope, c.Title, c.CreatedAt.Format("2006-01-02 15:04"))
		}
		_ = w.Flush()
		return nil
	},
}

var candidateApproveCmd = &cobra.Command{
	Use:   "approve <candidate-id>",
	Short: "Approve a memory candidate to become permanent memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.MemorySvc.ApproveCandidate(ctx, args[0]); err != nil {
			return err
		}
		fmt.Printf("Memory candidate %s approved.\n", args[0])
		return nil
	},
}

var candidateRejectCmd = &cobra.Command{
	Use:   "reject <candidate-id>",
	Short: "Reject a memory candidate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.MemorySvc.RejectCandidate(ctx, args[0]); err != nil {
			return err
		}
		fmt.Printf("Memory candidate %s rejected.\n", args[0])
		return nil
	},
}

func init() {
	candidateCmd.AddCommand(candidateListCmd)
	candidateCmd.AddCommand(candidateApproveCmd)
	candidateCmd.AddCommand(candidateRejectCmd)

	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(candidateCmd)
}
