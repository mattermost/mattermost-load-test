package main

import (
	"github.com/spf13/cobra"
	"os"
)

func showHelpIfNoFlags(cmd *cobra.Command, args []string) error {
	if cmd.Flags().NFlag() == 0 {
		cmd.Help()
		os.Exit(0)
	}

	return nil
}
