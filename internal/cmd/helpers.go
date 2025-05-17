package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/navaneethkn/cronocam/internal/auth"
	"github.com/navaneethkn/cronocam/internal/config"
	"github.com/navaneethkn/cronocam/internal/db"
	"github.com/navaneethkn/cronocam/internal/uploader"
)

// printPaths prints the absolute paths of important configuration files
func printPaths() error {
	credentialsPath, err := filepath.Abs(config.GetCredentialsPath())
	if err != nil {
		return fmt.Errorf("failed to get absolute credentials path: %v", err)
	}
	databasePath, err := filepath.Abs(config.GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to get absolute database path: %v", err)
	}

	fmt.Printf("Credentials path: %s\n", credentialsPath)
	fmt.Printf("Database path: %s\n\n", databasePath)
	return nil
}

func setupAuth() error {
	ctx := context.Background()
	authenticator, err := auth.New(config.GetCredentialsPath())
	if err != nil {
		return fmt.Errorf("failed to create authenticator: %v", err)
	}

	_, err = authenticator.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}

	return nil
}

func uploadPhotos(recursive, force bool, maxFiles int64) error {
	ctx := context.Background()

	// Initialize authenticator
	authenticator, err := auth.New(config.GetCredentialsPath())
	if err != nil {
		return fmt.Errorf("failed to create authenticator: %v", err)
	}

	// Get OAuth2 client
	client, err := authenticator.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}

	// Initialize database
	database, err := db.New(config.GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize uploader with configuration
	photoUploader, err := uploader.New(client, uploader.Config{
		ChunkSize:          config.GetChunkSize(),
		MaxRetries:         config.GetMaxRetries(),
		RequestsPerSecond:  config.GetRequestsPerSecond(),
		MaxBurst:           config.GetMaxBurst(),
	})
	if err != nil {
		return fmt.Errorf("failed to create uploader: %v", err)
	}

	// Track number of files uploaded
	var uploadCount int64

	// Walk function for processing files
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if we've hit the upload limit
		if maxFiles > 0 && uploadCount >= maxFiles {
			return filepath.SkipAll
		}

		// Skip directories if not recursive
		if info.IsDir() {
			if !recursive && path != config.GetUploadPath() {
				return filepath.SkipDir
			}
			return nil
		}

		if !photoUploader.IsSupportedFile(path) {
			return nil
		}

		// Calculate file hash
		hash, err := photoUploader.CalculateFileHash(path)
		if err != nil {
			log.Printf("Failed to calculate hash for %s: %v", path, err)
			return nil
		}

		// Check if file was already uploaded
		if !force {
			uploaded, err := database.IsFileUploaded(hash)
			if err != nil {
				log.Printf("Failed to check upload status for %s: %v", path, err)
				return nil
			}

			if uploaded {
				log.Printf("Skipping %s (already uploaded)", path)
				return nil
			}
		}

		// Upload file
		log.Printf("Uploading %s...", path)
		googleID, err := photoUploader.UploadFile(ctx, path)
		if err != nil {
			log.Printf("Failed to upload %s: %v", path, err)
			return nil
		}

		// Save to database
		err = database.SaveUploadedFile(&db.UploadedFile{
			FilePath: path,
			FileHash: hash,
			GoogleID: googleID,
		})
		if err != nil {
			log.Printf("Failed to save upload record for %s: %v", path, err)
			return nil
		}

		log.Printf("Successfully uploaded %s", path)
		uploadCount++

		// Check if we've hit the limit after successful upload
		if maxFiles > 0 && uploadCount >= maxFiles {
			log.Printf("Reached upload limit of %d files", maxFiles)
			return filepath.SkipAll
		}
		return nil
	}

	// Start walking the directory
	return filepath.Walk(config.GetUploadPath(), walkFn)
}
