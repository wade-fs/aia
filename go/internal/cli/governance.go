package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/domain/governance"
)

var (
	emergencyStopReason string
	resumeForce         bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop system activities or engage Emergency Stop (AIA-17)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// By default or when --emergency is supplied, trigger Emergency Stop
		ctx := context.Background()
		if appInst == nil {
			return fmt.Errorf("application not initialized")
		}

		actor := governance.Actor{Type: "HUMAN", ID: "cli_operator"}
		reason := emergencyStopReason
		if reason == "" {
			reason = "Emergency Stop triggered via CLI"
		}

		if appInst.WorkerMgr != nil {
			if err := appInst.WorkerMgr.TriggerEmergencyStop(ctx, actor, reason); err != nil {
				return fmt.Errorf("failed to engage emergency stop: %w", err)
			}
		} else if appInst.GovernanceSvc != nil {
			if err := appInst.GovernanceSvc.TriggerEmergencyStop(ctx, actor, reason); err != nil {
				return fmt.Errorf("failed to engage emergency stop: %w", err)
			}
		} else {
			return fmt.Errorf("governance runtime not available")
		}

		fmt.Println("🛑 EMERGENCY STOP TRIGGERED!")
		fmt.Printf("   Reason:    %s\n", reason)
		fmt.Println("   Task dispatch is immediately frozen (<1s).")
		fmt.Println("   All executing tasks have received cancellation signals.")
		fmt.Println("   Use 'aia resume --force' to unfreeze dispatch.")
		return nil
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume task dispatching from an Emergency Stop (Human Owner only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil {
			return fmt.Errorf("application not initialized")
		}

		actor := governance.Actor{Type: "HUMAN", ID: "cli_operator"}

		if appInst.WorkerMgr != nil {
			if err := appInst.WorkerMgr.ResumeDispatch(ctx, actor, resumeForce); err != nil {
				return fmt.Errorf("failed to resume runtime: %w", err)
			}
		} else if appInst.GovernanceSvc != nil {
			if err := appInst.GovernanceSvc.ResumeDispatch(ctx, actor, resumeForce); err != nil {
				return fmt.Errorf("failed to resume runtime: %w", err)
			}
		} else {
			return fmt.Errorf("governance runtime not available")
		}

		fmt.Println("✓ Runtime dispatch RESUMED to RUNNING state.")
		return nil
	},
}

func init() {
	stopCmd.Flags().StringVarP(&emergencyStopReason, "reason", "r", "", "Reason for emergency stop")
	resumeCmd.Flags().BoolVarP(&resumeForce, "force", "f", false, "Force resume even if prior condition requires verification")
}
