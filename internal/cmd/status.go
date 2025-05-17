package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/navaneethkn/cronocam/internal/config"
	"github.com/navaneethkn/cronocam/internal/db"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show upload status and statistics",
	Long: `Display upload status information including:
- Number of files uploaded to Google Photos
- Pending files to upload
- Any upload errors
- Last upload time`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Print paths
	if err := printPaths(); err != nil {
		return err
	}

	// Open database
	database, err := db.New(config.GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer database.Close()

	// Get statistics
	stats, err := database.GetUploadStats()
	if err != nil {
		return fmt.Errorf("failed to get upload statistics: %v", err)
	}

	// Get pending files
	pendingFiles, err := database.GetPendingFiles()
	if err != nil {
		return fmt.Errorf("failed to get pending files: %v", err)
	}

	// Get recent errors
	errors, err := database.GetRecentErrors()
	if err != nil {
		return fmt.Errorf("failed to get recent errors: %v", err)
	}

	// Format output
	fmt.Printf("Upload Status:\n")
	fmt.Printf("-------------\n")
	fmt.Printf("Total files uploaded: %d\n", stats.TotalUploaded)
	
	if stats.LastUploadTime != nil {
		relativeTime := formatRelativeTime(*stats.LastUploadTime)
		fmt.Printf("Last upload: %s\n", relativeTime)
	} else {
		fmt.Printf("Last upload: Never\n")
	}

	fmt.Printf("\nPending Files: %d\n", len(pendingFiles))
	if len(pendingFiles) > 0 {
		fmt.Printf("First 5 pending files:\n")
		for i, file := range pendingFiles {
			if i >= 5 {
				break
			}
			fmt.Printf("- %s\n", filepath.Base(file))
		}
	}

	if len(errors) > 0 {
		fmt.Printf("\nRecent Errors:\n")
		for _, err := range errors {
			fmt.Printf("- %s: %s\n", filepath.Base(err.File), err.Message)
		}
	}

	return nil
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d minute%s ago", mins, pluralize(mins))
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	default:
		return t.Format("Jan 2, 2006")
	}
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
