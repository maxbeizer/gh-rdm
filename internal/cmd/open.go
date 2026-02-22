package cmd

import (
	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open [url]",
		Short: "Open a URL on the host machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			_, err := c.SendCommand(cmd.Context(), "open", args[0])
			return err
		},
	}
}
