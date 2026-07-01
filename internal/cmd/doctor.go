package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

const rdmTunnelPort = "7391"

type doctorDeps struct {
	socketPath func() string
	statSocket func(string) error
	statusUnix func(context.Context, string) error
	statusTCP  func(context.Context, string) error
	getenv     func(string) string
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose gh-rdm server and tunnel connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.Context(), cmd.OutOrStdout(), defaultDoctorDeps())
		},
	}
}

func defaultDoctorDeps() doctorDeps {
	return doctorDeps{
		socketPath: client.UnixSocketPath,
		statSocket: func(path string) error {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSocket == 0 {
				return fmt.Errorf("exists but is not a unix socket")
			}
			return nil
		},
		statusUnix: func(ctx context.Context, socketPath string) error {
			return checkStatus(ctx, client.NewWithSocketPath(socketPath))
		},
		statusTCP: func(ctx context.Context, address string) error {
			return checkStatus(ctx, client.NewWithTCPAddress(address))
		},
		getenv: os.Getenv,
	}
}

func runDoctor(ctx context.Context, out io.Writer, deps doctorDeps) error {
	socketPath := deps.socketPath()
	failures := 0
	remote := isRemoteEnvironment(deps.getenv)

	fmt.Fprintln(out, "gh-rdm doctor")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Unix socket (this machine)")
	if remote {
		fmt.Fprintln(out, "  - skipped (running in SSH or Codespaces environment; checking TCP tunnel instead)")
	} else {
		if printCheck(out, "socket path exists", socketPath, deps.statSocket(socketPath)) {
			failures++
		}
		if printCheck(out, "server responds over unix socket", socketPath, deps.statusUnix(ctx, socketPath)) {
			failures++
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Remote tunnel")
	if remote {
		addresses := []string{
			"localhost:" + rdmTunnelPort,
			"127.0.0.1:" + rdmTunnelPort,
			"[::1]:" + rdmTunnelPort,
		}
		remoteFailures := 0
		for _, address := range addresses {
			if printCheck(out, "server responds over tcp", address, deps.statusTCP(ctx, address)) {
				failures++
				remoteFailures++
			}
		}
		if remoteFailures > 0 {
			printRepairCommand(out, deps.getenv)
		}
	} else {
		fmt.Fprintln(out, "  - skipped (not running in SSH or Codespaces environment)")
	}

	if failures > 0 {
		return fmt.Errorf("gh-rdm doctor found %d issue(s)", failures)
	}
	return nil
}

func printCheck(out io.Writer, name, detail string, err error) bool {
	if err == nil {
		fmt.Fprintf(out, "  ✓ %s: %s\n", name, detail)
		return false
	}
	fmt.Fprintf(out, "  ✗ %s: %s (%v)\n", name, detail, err)
	return true
}

func printRepairCommand(out io.Writer, getenv func(string) string) {
	fmt.Fprintln(out)
	if isCodespaceEnvironment(getenv) {
		codespace := getenv("CODESPACE_NAME")
		if codespace == "" {
			codespace = "<codespace>"
		}
		fmt.Fprintln(out, "Repair command (run on your local machine):")
		fmt.Fprintf(out, "  gh cs ssh -c %s -- -o ExitOnForwardFailure=yes -N -R localhost:%s:$(gh rdm socket)\n", codespace, rdmTunnelPort)
		return
	}

	fmt.Fprintln(out, "Repair command (run on your local machine, replacing <host>):")
	fmt.Fprintf(out, "  ssh -o ExitOnForwardFailure=yes -N -R localhost:%s:$(gh rdm socket) <host>\n", rdmTunnelPort)
}

func isRemoteEnvironment(getenv func(string) string) bool {
	return getenv("SSH_TTY") != "" ||
		getenv("SSH_CLIENT") != "" ||
		getenv("SSH_CONNECTION") != "" ||
		isCodespaceEnvironment(getenv)
}

func isCodespaceEnvironment(getenv func(string) string) bool {
	return getenv("CODESPACES") == "true" || getenv("CODESPACE_NAME") != ""
}

func checkStatus(ctx context.Context, c *client.Client) error {
	data, err := c.SendCommand(ctx, "status")
	if err != nil {
		return err
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("parsing status response: %w", err)
	}
	if response.Status != "running" {
		return errors.New("status response did not report running")
	}
	return nil
}
