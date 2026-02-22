package cmd

import (
	"fmt"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

func newPasteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "paste",
		Short: "Paste clipboard content to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New()
			result, err := c.SendCommand(cmd.Context(), "paste")
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), string(result))
			return nil
		},
	}
}
