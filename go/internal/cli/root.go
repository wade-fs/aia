package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/app"
)

var (
	cfgPath string
	appInst *app.App
)

var RootCmd = &cobra.Command{
	Use:   "aia",
	Short: "AIA is a Personal AI Agent Platform running on your desktop",
	Long: `AIA (Autonomous Intelligence Architecture) is a Personal AI Agent Platform.
It coordinates domain agents, persists long-term memories, manages task execution,
and integrates with Antigravity CLI 1.4.12.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Doctor or version commands can run without full app if desired, but we can init app
		if cmd.Name() == "version" {
			return nil
		}
		a, err := app.NewApp(cfgPath, nil)
		if err != nil {
			return fmt.Errorf("failed to initialize AIA application: %w", err)
		}
		appInst = a
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if appInst != nil {
			_ = appInst.Close()
		}
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "path to config file (default ~/.config/aia/config.yaml or ./configs/aia.yaml)")

	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(doctorCmd)
	RootCmd.AddCommand(chatCmd)
	RootCmd.AddCommand(taskCmd)
	RootCmd.AddCommand(agentCmd)
	RootCmd.AddCommand(projectCmd)
	RootCmd.AddCommand(memoryCmd)
	RootCmd.AddCommand(executionCmd)
	RootCmd.AddCommand(investmentCmd)
	RootCmd.AddCommand(scheduleCmd)
	RootCmd.AddCommand(workerCmd)
	RootCmd.AddCommand(monitorCmd)
	RootCmd.AddCommand(decisionCmd)
	RootCmd.AddCommand(approvalCmd)
	RootCmd.AddCommand(stopCmd)
	RootCmd.AddCommand(resumeCmd)
	RootCmd.AddCommand(cycleCmd)
	RootCmd.AddCommand(goalCmd)
	RootCmd.AddCommand(learningCmd)
	RootCmd.AddCommand(experimentCmd)
	RootCmd.AddCommand(delegationCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print AIA version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("AIA Personal Agent v1.0 (Go 1.24+ / Antigravity 1.4.12)")
	},
}
