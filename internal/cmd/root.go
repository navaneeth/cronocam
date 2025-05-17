package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/navaneethkn/cronocam/internal/config"
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
			if err := config.Initialize(cfgFile); err != nil {
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
	// Config initialization is handled in config.Initialize
}
