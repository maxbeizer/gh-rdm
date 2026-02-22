package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

func Execute(ctx context.Context, userMessages *log.Logger) error {
	rootCmd := &cobra.Command{
		Use:   "gh-rdm",
		Short: "Remote Development Manager - clipboard and open forwarding over SSH",
	}

	rootCmd.AddCommand(
		newServerCmd(userMessages),
		newStopCmd(),
		newCopyCmd(),
		newPasteCmd(),
		newOpenCmd(),
		newSocketCmd(),
	)

	return rootCmd.ExecuteContext(ctx)
}
