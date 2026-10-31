package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage and inspect AIA domain agents",
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		agents, err := appInst.AgentSvc.List(ctx)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDOMAIN\tENABLED\tCAPABILITIES")
		for _, a := range agents {
			enabledStr := "yes"
			if !a.Enabled {
				enabledStr = "no"
			}
			caps := strings.Join(a.Capabilities, ",")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Domain, enabledStr, caps)
		}
		_ = w.Flush()
		return nil
	},
}

var agentShowCmd = &cobra.Command{
	Use:   "show <agent-id>",
	Short: "Show details for an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		a, err := appInst.AgentSvc.Get(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Println("=== Agent Details ===")
		fmt.Printf("ID:           %s\n", a.ID)
		fmt.Printf("Name:         %s\n", a.Name)
		fmt.Printf("Domain:       %s\n", a.Domain)
		fmt.Printf("Enabled:      %v\n", a.Enabled)
		fmt.Printf("Capabilities: %s\n", strings.Join(a.Capabilities, ", "))
		fmt.Printf("Description:  %s\n\n", a.Description)
		fmt.Printf("Instructions:\n%s\n", a.Instructions)
		return nil
	},
}

var agentEnableCmd = &cobra.Command{
	Use:   "enable <agent-id>",
	Short: "Enable an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.AgentSvc.SetEnabled(ctx, args[0], true); err != nil {
			return err
		}
		fmt.Printf("Agent %s enabled.\n", args[0])
		return nil
	},
}

var agentDisableCmd = &cobra.Command{
	Use:   "disable <agent-id>",
	Short: "Disable an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.AgentSvc.SetEnabled(ctx, args[0], false); err != nil {
			return err
		}
		fmt.Printf("Agent %s disabled.\n", args[0])
		return nil
	},
}

func init() {
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentShowCmd)
	agentCmd.AddCommand(agentEnableCmd)
	agentCmd.AddCommand(agentDisableCmd)
}
