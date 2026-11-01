package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/experiment"
)

var experimentCmd = &cobra.Command{
	Use:   "experiment",
	Short: "Manage longitudinal & generational autonomous experiments (Phase 5)",
}

var expCreateDualCmd = &cobra.Command{
	Use:   "create-dual",
	Short: "Create a Generational Investment Decision Experiment (G1 frozen rule baseline vs G2 LLM)",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString("project")
		goalID, _ := cmd.Flags().GetString("goal")
		name, _ := cmd.Flags().GetString("name")
		capital, _ := cmd.Flags().GetInt64("capital")
		modelID, _ := cmd.Flags().GetString("model")

		bundle, err := appInst.ExperimentCtrl.CreateDualExperiment(context.Background(), experiment.CreateDualInput{
			ProjectID:      projectID,
			PrimaryGoalID:  goalID,
			Name:           name,
			InitialCapital: capital,
			G2ModelID:      modelID,
		})
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(bundle, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var expListCmd = &cobra.Command{
	Use:   "list",
	Short: "List autonomous experiments",
	RunE: func(cmd *cobra.Command, args []string) error {
		domainFilter, _ := cmd.Flags().GetString("domain")
		exps, err := appInst.ExperimentRepo.ListExperiments(context.Background(), domainFilter, 50)
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(exps, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var expGenerationsCmd = &cobra.Command{
	Use:   "generations <experiment-id>",
	Short: "List all generational cohorts (G1, G2, G2.x) in an experiment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gens, err := appInst.ExperimentRepo.ListGenerations(context.Background(), args[0])
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(gens, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var expStepCmd = &cobra.Command{
	Use:   "step <experiment-id>",
	Short: "Step an observation day across G1 and active G2/G2.x cohorts (Zero Look-Ahead)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		symbol, _ := cmd.Flags().GetString("symbol")
		asOfStr, _ := cmd.Flags().GetString("as-of")
		asOf := time.Now().UTC()
		if asOfStr != "" {
			parsed, err := time.Parse(time.RFC3339, asOfStr)
			if err != nil {
				return fmt.Errorf("invalid --as-of RFC3339 timestamp: %w", err)
			}
			asOf = parsed
		}
		res, err := appInst.ExperimentCtrl.StepObservation(context.Background(), experiment.StepInput{
			ExperimentID: args[0],
			Symbol:       symbol,
			AsOf:         asOf,
		})
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var expCompareCmd = &cobra.Command{
	Use:   "compare <experiment-id>",
	Short: "Compare G1 baseline against a candidate generation (G2 or G2.x) across Track A & Track B",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		baseline, _ := cmd.Flags().GetString("baseline")
		candidate, _ := cmd.Flags().GetString("candidate")
		asOfStr, _ := cmd.Flags().GetString("as-of")
		asOf := time.Now().UTC()
		if asOfStr != "" {
			parsed, err := time.Parse(time.RFC3339, asOfStr)
			if err != nil {
				return fmt.Errorf("invalid --as-of RFC3339 timestamp: %w", err)
			}
			asOf = parsed
		}
		cmp, err := appInst.ExperimentCtrl.CompareGenerations(context.Background(), args[0], baseline, candidate, asOf)
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(cmp, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	expCreateDualCmd.Flags().String("project", "default", "project ID")
	expCreateDualCmd.Flags().String("goal", "", "primary strategic goal ID")
	expCreateDualCmd.Flags().String("name", "", "experiment name")
	expCreateDualCmd.Flags().Int64("capital", 100000, "initial virtual capital in TWD for each generation")
	expCreateDualCmd.Flags().String("model", "gemini-default", "G2 model identifier")

	expListCmd.Flags().String("domain", "", "optional domain filter (e.g. investment)")

	expStepCmd.Flags().String("symbol", "2330", "target stock symbol")
	expStepCmd.Flags().String("as-of", "", "point-in-time RFC3339 timestamp")

	expCompareCmd.Flags().String("baseline", "G1", "baseline generation code")
	expCompareCmd.Flags().String("candidate", "G2", "candidate generation code")
	expCompareCmd.Flags().String("as-of", "", "point-in-time RFC3339 timestamp")

	experimentCmd.AddCommand(expCreateDualCmd)
	experimentCmd.AddCommand(expListCmd)
	experimentCmd.AddCommand(expGenerationsCmd)
	experimentCmd.AddCommand(expStepCmd)
	experimentCmd.AddCommand(expCompareCmd)
}
