package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	backupNote   string
	restoreForce bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage AIA database, configurations, and state backups (AIA-19 §12)",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an online point-in-time backup snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.BackupSvc == nil {
			return fmt.Errorf("backup service not initialized")
		}
		manifest, err := appInst.BackupSvc.CreateBackup(context.Background(), backupNote)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		fmt.Printf("✅ Backup successfully created:\n")
		fmt.Printf("  • Backup ID:       %s\n", manifest.BackupID)
		fmt.Printf("  • Created At:      %s\n", manifest.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("  • Database File:   %s (%d bytes)\n", manifest.DatabaseFile, manifest.DatabaseSize)
		fmt.Printf("  • SHA-256:         %s\n", manifest.ChecksumSHA256)
		fmt.Printf("  • Schema Version:  %d\n", manifest.SchemaVersion)
		if manifest.Note != "" {
			fmt.Printf("  • Note:            %s\n", manifest.Note)
		}
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all existing backup snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.BackupSvc == nil {
			return fmt.Errorf("backup service not initialized")
		}
		list, err := appInst.BackupSvc.ListBackups(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}
		if len(list) == 0 {
			fmt.Println("No backups found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "BACKUP ID\tCREATED AT\tSCHEMA\tSIZE (BYTES)\tSTATUS\tNOTE")
		fmt.Fprintln(w, "---------\t----------\t------\t------------\t------\t----")
		for _, m := range list {
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
				m.BackupID,
				m.CreatedAt.Format("2006-01-02 15:04:05"),
				m.SchemaVersion,
				m.DatabaseSize,
				m.Status,
				m.Note,
			)
		}
		w.Flush()
		return nil
	},
}

var backupVerifyCmd = &cobra.Command{
	Use:   "verify <backup-id>",
	Short: "Verify checksum and SQLite integrity of a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.BackupSvc == nil {
			return fmt.Errorf("backup service not initialized")
		}
		backupID := args[0]
		res, err := appInst.BackupSvc.VerifyBackup(context.Background(), backupID)
		if err != nil {
			return fmt.Errorf("verification error: %w", err)
		}

		if res.Valid {
			fmt.Printf("✅ Backup [%s] is VALID and INTACT:\n", backupID)
			fmt.Printf("  • Checksum SHA-256: MATCHED\n")
			fmt.Printf("  • SQLite Integrity: OK\n")
			fmt.Printf("  • Schema Version:   %d\n", res.SchemaVersion)
		} else {
			fmt.Printf("❌ Backup [%s] FAILED verification:\n", backupID)
			for _, issue := range res.Issues {
				fmt.Printf("  - %s\n", issue)
			}
		}
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "Restore database state from a verified backup (AIA-19 §12.4)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.BackupSvc == nil {
			return fmt.Errorf("backup service not initialized")
		}
		backupID := args[0]
		targetDB := appInst.Config.Database.Path
		if targetDB == "" {
			return fmt.Errorf("target database path not configured in config.yaml")
		}

		if !restoreForce {
			fmt.Printf("⚠️  WARNING: Restoring backup %s will overwrite %s.\n", backupID, targetDB)
			fmt.Printf("Run with --force to confirm restoration.\n")
			return nil
		}

		if err := appInst.BackupSvc.RestoreBackup(context.Background(), backupID, targetDB); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		fmt.Printf("✅ Backup [%s] successfully restored to %s.\n", backupID, targetDB)
		fmt.Printf("A pre-restore safety copy was preserved.\n")
		return nil
	},
}

func init() {
	backupCreateCmd.Flags().StringVarP(&backupNote, "note", "n", "", "Optional human note describing backup")
	backupRestoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "Force restore and overwrite target database")

	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupVerifyCmd)
	backupCmd.AddCommand(backupRestoreCmd)

	RootCmd.AddCommand(backupCmd)
}
