package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/navaneethkn/cronocam/internal/config"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [directory]",
	Short: "Upload photos from a directory",
	Long: `Upload photos from a directory to Google Photos.
Recursively searches for supported image files and uploads them.
Tracks uploaded files to avoid duplicates.`,
	Args: cobra.ExactArgs(1),
	RunE: runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// Add flags
	uploadCmd.Flags().BoolP("recursive", "r", true, "recursively search for files in subdirectories")
	uploadCmd.Flags().BoolP("force", "f", false, "force upload even if file was previously uploaded")
}

func runUpload(cmd *cobra.Command, args []string) error {
	// Get directory path from args
	dirPath := args[0]

	// Get flags
	recursive, _ := cmd.Flags().GetBool("recursive")
	force, _ := cmd.Flags().GetBool("force")

	// Convert to absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Validate upload directory
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to access directory %s: %v", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absPath)
	}

	// Validate credentials file first
	credentialsPath, err := filepath.Abs(config.GetCredentialsPath())
	if err != nil {
		return fmt.Errorf("failed to get absolute credentials path: %v", err)
	}
	info, err = os.Stat(credentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("credentials file not found at %s - please create it first", credentialsPath)
		}
		return fmt.Errorf("failed to access credentials file: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, expected a file", credentialsPath)
	}

	// Print paths
	if err := printPaths(); err != nil {
		return err
	}

	// Set upload path in config
	config.SetUploadPath(absPath)

	// Ensure required directories exist
	if err := config.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	// Start upload process
	return uploadPhotos(recursive, force)
}
