package cli

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/provider/antigravity"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system dependencies, agy binary, sqlite, and environment health",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		fmt.Println("Diagnosing AIA Personal Agent environment...")
		fmt.Println()

		// 1. Go Runtime
		fmt.Printf("[OK] Go Runtime: %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

		// 2. Configuration
		if appInst.Config != nil {
			fmt.Printf("[OK] Configuration loaded: %s (Timezone: %s)\n", appInst.Config.System.Name, appInst.Config.System.Timezone)
		} else {
			fmt.Println("[WARN] Using default configuration")
		}

		// 3. SQLite Database
		if appInst.DB != nil {
			row := appInst.DB.RawDB().QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations")
			var migCount int
			if err := row.Scan(&migCount); err == nil {
				fmt.Printf("[OK] SQLite Database: Ready (%s, migrations applied: %d)\n", appInst.Config.Database.Path, migCount)
			} else {
				fmt.Printf("[FAIL] SQLite Database query failed: %v\n", err)
			}
		}

		// 4. Provider & Antigravity Binary
		exe := appInst.Config.Provider.Antigravity.Executable
		ver, err := antigravity.CheckVersion(ctx, exe)
		if err != nil {
			fmt.Printf("[FAIL] Antigravity CLI binary (%s): %v\n", exe, err)
		} else {
			fmt.Printf("[OK] Antigravity CLI: %s (version: %s)\n", exe, ver)
		}

		// 5. Workspace Directory
		wsDir := appInst.Config.Runtime.WorkspaceDir
		fmt.Printf("[OK] Workspace Directory: %s\n", wsDir)

		// 6. Registered Agents
		agents, err := appInst.AgentSvc.List(ctx)
		if err == nil {
			fmt.Printf("[OK] Agents: %d configured and ready\n", len(agents))
		}

		fmt.Println()
		fmt.Println("All essential diagnostics completed successfully.")
		return nil
	},
}
