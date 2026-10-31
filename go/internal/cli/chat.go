package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/orchestrator"
)

var (
	chatProjectID string
	chatJSON      bool
)

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Chat with the AIA Personal Agent or dispatch tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Single prompt mode
		if len(args) > 0 {
			prompt := strings.Join(args, " ")
			return processChatInput(ctx, prompt)
		}

		// Interactive REPL mode
		fmt.Println("🤖 AIA Personal Agent v1 (Type 'exit' or 'quit' to leave)")
		fmt.Println("---------------------------------------------------------")
		scanner := bufio.NewScanner(os.Stdin)

		for {
			fmt.Print("\nYOU > ")
			if !scanner.Scan() {
				break
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if line == "exit" || line == "quit" {
				fmt.Println("Goodbye!")
				break
			}

			if err := processChatInput(ctx, line); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
		return nil
	},
}

func processChatInput(ctx context.Context, text string) error {
	resp, err := appInst.PersonalAgent.HandleInput(ctx, orchestrator.UserInput{
		Content:   text,
		ProjectID: chatProjectID,
	})
	if err != nil {
		return err
	}

	if chatJSON {
		data, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("\nAIA > %s\n", resp.Content)
	return nil
}

func init() {
	chatCmd.Flags().StringVarP(&chatProjectID, "project", "p", "", "project ID for conversation context")
	chatCmd.Flags().BoolVar(&chatJSON, "json", false, "output response in structured JSON")
}
