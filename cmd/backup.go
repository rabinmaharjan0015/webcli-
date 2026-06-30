package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/yenya/webcli/internal/store"
)

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupListCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore shared memory",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a backup of the shared memory store",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New(cfg.Store.MemoryFile)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		backupDir := cfg.Store.BackupDir
		if backupDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("home dir: %w", err)
			}
			backupDir = filepath.Join(home, ".webcli", "backups")
		}

		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("create backup dir: %w", err)
		}

		timestamp := time.Now().Format("20060102_150405")
		src := s.Path()
		dst := filepath.Join(backupDir, fmt.Sprintf("memory_%s.json", timestamp))

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}

		fmt.Printf("Backup created: %s (%d items)\n", dst, s.Count())
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore shared memory from a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupPath := args[0]

		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", backupPath)
		}

		s, err := store.New(cfg.Store.MemoryFile)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}

		if err := copyFile(backupPath, s.Path()); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		// Reload to verify
		s2, err := store.New(cfg.Store.MemoryFile)
		if err != nil {
			return fmt.Errorf("verify restored store: %w", err)
		}

		fmt.Printf("Restored from %s (%d items)\n", backupPath, s2.Count())
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		backupDir := cfg.Store.BackupDir
		if backupDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("home dir: %w", err)
			}
			backupDir = filepath.Join(home, ".webcli", "backups")
		}

		entries, err := os.ReadDir(backupDir)
		if os.IsNotExist(err) {
			fmt.Println("No backups found.")
			return nil
		}
		if err != nil {
			return fmt.Errorf("read backup dir: %w", err)
		}

		var count int
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				info, _ := e.Info()
				size := info.Size()
				fmt.Printf("  %s (%d bytes)\n", e.Name(), size)
				count++
			}
		}

		if count == 0 {
			fmt.Println("No backups found.")
		}
		return nil
	},
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
