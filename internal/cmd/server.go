package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/maxbeizer/gh-rdm/internal/hostservice"
	"github.com/maxbeizer/gh-rdm/internal/server"
	"github.com/spf13/cobra"
)

func newServerCmd(userMessages *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the gh-rdm server",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			logDir := filepath.Join(homeDir, ".gh-rdm")
			if err := os.MkdirAll(logDir, 0o755); err != nil {
				return err
			}

			logFile, err := os.OpenFile(filepath.Join(logDir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			defer logFile.Close()

			svc := &hostservice.Service{}
			socketPath := client.UnixSocketPath()
			srv := server.New(svc, socketPath, userMessages)

			return srv.Listen(cmd.Context())
		},
	}
}
