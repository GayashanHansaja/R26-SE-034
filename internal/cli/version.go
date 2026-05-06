package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of bridgectl",
	Long:  `All software has versions. This is bridgectl's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bridgectl version %s\n", RootCmd.Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
