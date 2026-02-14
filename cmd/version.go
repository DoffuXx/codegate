package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display the version, commit hash, and build date of CodeGate.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("codegate version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
