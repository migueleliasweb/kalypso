package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Kalypso CLI",
	Long:  `All software has versions. This is Kalypso CLI's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v1alpha2")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
