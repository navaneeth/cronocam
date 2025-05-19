package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navaneethkn/cronocam/internal/config"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [directory]",
	Short: "Upload photos from a directory",
	Long: `Upload photos from a directory to Google Photos.
Recursively searches for supported image files and uploads them.
Tracks uploaded files to avoid duplicates.

You can also provide a text file containing a list of file paths to upload
using the --file-list option. Each line in the file should be a full path
to a photo or video file.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// Add flags
	uploadCmd.Flags().BoolP("recursive", "r", true, "recursively search for files in subdirectories")
	uploadCmd.Flags().Int64P("max-files", "m", 0, "maximum number of files to upload (0 for unlimited)")
	uploadCmd.Flags().BoolP("force", "f", false, "force upload even if file was previously uploaded")
	uploadCmd.Flags().StringP("file-list", "l", "", "path to text file containing list of files to upload")
}

func runUpload(cmd *cobra.Command, args []string) error {
	// Get flags
	var recursive bool
	var force bool
	var maxFiles int64
	var fileList string
	recursive, _ = cmd.Flags().GetBool("recursive")
	force, _ = cmd.Flags().GetBool("force")
	maxFiles, _ = cmd.Flags().GetInt64("max-files")
	fileList, _ = cmd.Flags().GetString("file-list")

	// Check if using file list
	if fileList != "" {
		// Read file paths from text file
		fileData, err := os.ReadFile(fileList)
		if err != nil {
			return fmt.Errorf("failed to read file list: %v", err)
		}

		// Split into lines and filter empty lines
		var files []string
		for _, line := range strings.Split(string(fileData), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				files = append(files, line)
			}
		}

		// Validate all files exist before starting
		for _, file := range files {
			if _, err := os.Stat(file); err != nil {
				return fmt.Errorf("file not found: %s", file)
			}
		}

		// Print paths
		if err := printPaths(); err != nil {
			return err
		}

		// Start upload process
		return uploadFiles(files, force, maxFiles)
	}

	// Using directory mode
	if len(args) == 0 {
		return fmt.Errorf("directory path required when not using --file-list")
	}

	// Get directory path from args
	dirPath := args[0]

	// These variables were already declared at the top
	recursive, _ = cmd.Flags().GetBool("recursive")
	force, _ = cmd.Flags().GetBool("force")
	maxFiles, _ = cmd.Flags().GetInt64("max-files")

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
	return uploadPhotos(recursive, force, maxFiles)
}
