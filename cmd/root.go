package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kalypso",
	Short: "Kalypso CLI is a tool to manage and package Kalypso capabilities",
	Long:  `Kalypso CLI provides tools for working with Kalypso custom resource definitions, including converting Helm charts to Kalypso CRD templates.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
