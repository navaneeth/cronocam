package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/navaneethkn/cronocam/internal/config"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup Google Photos authentication",
	Long: `Setup Google Photos authentication by performing OAuth2 flow.
This will open your browser for authentication and save the credentials.`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Ensure required directories exist
	if err := config.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	if err := setupAuth(); err != nil {
		return err
	}

	fmt.Println("Setup completed successfully!")
	return nil
}
