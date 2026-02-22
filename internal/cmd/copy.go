package cmd

import (
	"io"
	"os"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

func newCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copy",
		Short: "Copy stdin content to clipboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			c := client.New()
			_, err = c.SendCommand(cmd.Context(), "copy", string(data))
			return err
		},
	}
}
