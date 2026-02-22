package cmd

import (
	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the gh-rdm server",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			_, err := c.SendCommand(cmd.Context(), "stop")
			return err
		},
	}
}
