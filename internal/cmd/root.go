package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/navaneethkn/cronocam/internal/config"
	"github.com/spf13/viper"
)

// Version information (set by build)
var Version = "dev"

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:     "cronocam",
		Version: Version,
		Short:   "A tool for uploading photos to Google Photos",
		Long: `cronocam is a command-line tool for uploading photos to Google Photos.
It supports batch uploading, resumable uploads, and tracks uploaded files
to avoid duplicates.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration before any command runs
			if err := config.Initialize(); err != nil {
				return fmt.Errorf("failed to initialize config: %v", err)
			}
			return nil
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in default locations
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		
		// Add user config directory
		if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
			viper.AddConfigPath(fmt.Sprintf("%s/photos-uploader", configHome))
		} else if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(fmt.Sprintf("%s/.config/photos-uploader", home))
		}
	}

	// Environment variables
	viper.SetEnvPrefix("PHOTOS")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}
}
