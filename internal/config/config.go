package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Default configuration values
const (
	DefaultCredentialsPath = "config/credentials.json"
	DefaultDatabasePath    = "data/uploads.db"
	DefaultChunkSize       = 5 * 1024 * 1024 // 5MB
	DefaultMaxRetries      = 3
	DefaultReqPerSec       = 5
	DefaultMaxBurst        = 10

	// Default supported file formats
	DefaultSupportedImages = ".jpg,.jpeg,.png,.gif,.heic,.heif,.webp,.tiff,.tif,.bmp"
	DefaultSupportedVideos = ".mpg,.mpeg,.avi,.mov,.mp4,.m4v,.wmv,.3gp,.3g2,.mkv,.mts,.m2ts"
)

var (
	v    *viper.Viper
	once sync.Once
)

// Initialize sets up the viper configuration
// GetSupportedFormats returns a map of supported file extensions
func GetSupportedFormats() map[string]bool {
	// Get configured formats
	imageFormats := strings.Split(v.GetString("supported_images"), ",")
	videoFormats := strings.Split(v.GetString("supported_videos"), ",")

	// Build map of supported formats
	supported := make(map[string]bool)
	for _, ext := range append(imageFormats, videoFormats...) {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			supported[ext] = true
		}
	}

	return supported
}

func Initialize() error {
	var initErr error
	once.Do(func() {
		v = viper.New()

		// Set defaults
		v.SetDefault("credentials_path", DefaultCredentialsPath)
		v.SetDefault("database_path", DefaultDatabasePath)
		v.SetDefault("chunk_size", DefaultChunkSize)
		v.SetDefault("max_retries", DefaultMaxRetries)
		v.SetDefault("rate_limit.requests_per_second", DefaultReqPerSec)
		v.SetDefault("rate_limit.max_burst", DefaultMaxBurst)
		v.SetDefault("supported_images", DefaultSupportedImages)
		v.SetDefault("supported_videos", DefaultSupportedVideos)

		// Environment variables
		v.SetEnvPrefix("PHOTOS")
		v.AutomaticEnv()

		// Config file
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				initErr = fmt.Errorf("could not find home directory: %v", err)
				return
			}
			configHome = filepath.Join(home, ".config")
		}

		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath(filepath.Join(configHome, "photos-uploader"))

		// Try to read config file, but don't fail if not found
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found, using defaults
				fmt.Printf("No config file found, using default values\n")
			} else {
				// Config file found but has errors
				fmt.Printf("Warning: Error in config file: %v\nUsing default values\n", err)
			}
		} else {
			fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
		}
	})

	return initErr
}

// GetCredentialsPath returns the configured credentials path
func GetCredentialsPath() string {
	return v.GetString("credentials_path")
}

// GetDatabasePath returns the configured database path
func GetDatabasePath() string {
	return v.GetString("database_path")
}

// GetUploadPath returns the configured upload path
func GetUploadPath() string {
	return v.GetString("upload_path")
}

// SetUploadPath sets the upload path
func SetUploadPath(path string) {
	v.Set("upload_path", path)
}

// GetChunkSize returns the configured chunk size
func GetChunkSize() int64 {
	return v.GetInt64("chunk_size")
}

// GetMaxRetries returns the configured max retries
func GetMaxRetries() int {
	return v.GetInt("max_retries")
}

// GetRequestsPerSecond returns the configured requests per second limit
func GetRequestsPerSecond() int {
	return v.GetInt("rate_limit.requests_per_second")
}

// GetMaxBurst returns the configured max burst limit
func GetMaxBurst() int {
	return v.GetInt("rate_limit.max_burst")
}

// EnsureDirectories creates necessary directories for credentials and database
func EnsureDirectories() error {
	dirs := []string{
		filepath.Dir(GetCredentialsPath()),
		filepath.Dir(GetDatabasePath()),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}
