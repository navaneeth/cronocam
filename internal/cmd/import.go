package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/navaneethkn/cronocam/internal/config"
	"github.com/navaneethkn/cronocam/internal/db"
	"github.com/navaneethkn/cronocam/internal/uploader"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [directory]",
	Short: "Import photos without uploading",
	Long: `Import photos from a directory without uploading them to Google Photos.
This command calculates file hashes and stores them in the database,
marking them as already uploaded so they won't be uploaded in the future.`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	// Add flags
	importCmd.Flags().BoolP("recursive", "r", true, "recursively search for files in subdirectories")
}

func runImport(cmd *cobra.Command, args []string) error {
	// Get directory path from args
	dirPath := args[0]

	// Get flags
	recursive, _ := cmd.Flags().GetBool("recursive")

	// Convert to absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Validate directory
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to access directory %s: %v", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absPath)
	}

	// Print paths
	if err := printPaths(); err != nil {
		return err
	}

	// Initialize database
	database, err := db.New(config.GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create uploader just for file type validation and hash calculation
	// We don't need rate limiting for import, but we need valid config values
	u, err := uploader.New(nil, uploader.Config{
		ChunkSize:         config.GetChunkSize(),
		MaxRetries:        config.GetMaxRetries(),
		RequestsPerSecond: 1,  // Doesn't matter for import
		MaxBurst:          1,  // Doesn't matter for import
	})
	if err != nil {
		return fmt.Errorf("failed to create uploader: %v", err)
	}

	// Walk function for processing files
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories if not recursive
		if info.IsDir() {
			if !recursive && path != absPath {
				return filepath.SkipDir
			}
			return nil
		}

		if !u.IsSupportedFile(path) {
			return nil
		}

		// Calculate file hash
		hash, err := u.CalculateFileHash(path)
		if err != nil {
			log.Printf("Failed to calculate hash for %s: %v", path, err)
			return nil
		}

		// Check if file was already imported
		imported, err := database.IsFileUploaded(hash)
		if err != nil {
			log.Printf("Failed to check import status for %s: %v", path, err)
			return nil
		}

		if imported {
			log.Printf("Skipping %s (already imported)", path)
			return nil
		}

		// Save to database with empty GoogleID (since we're not uploading)
		err = database.SaveUploadedFile(&db.UploadedFile{
			FilePath: path,
			FileHash: hash,
			GoogleID: "", // Empty since we're not uploading
		})
		if err != nil {
			log.Printf("Failed to save import record for %s: %v", path, err)
			return nil
		}

		log.Printf("Successfully imported %s", path)
		return nil
	}

	// Start walking the directory
	return filepath.Walk(absPath, walkFn)
}
