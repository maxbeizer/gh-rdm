package cmd

import (
	"fmt"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

func newSocketCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "socket",
		Short: "Print the unix socket path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), client.UnixSocketPath())
		},
	}
}
