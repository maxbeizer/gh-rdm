package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

type codespace struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type tunnelDeps struct {
	socketPath     func() string
	statusUnix     func(context.Context, string) error
	startServer    func() error
	listCodespaces func(context.Context) ([]codespace, error)
	runTunnel      func(context.Context, string, string) error
}

func newTunnelCmd() *cobra.Command {
	var codespaceName string

	cmd := &cobra.Command{
		Use:   "tunnel [codespace]",
		Short: "Start a GitHub Codespaces tunnel to the local gh-rdm server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if codespaceName != "" {
					return fmt.Errorf("provide a codespace either as an argument or with --codespace, not both")
				}
				codespaceName = args[0]
			}
			return runTunnel(cmd.Context(), cmd.OutOrStdout(), codespaceName, defaultTunnelDeps())
		},
	}

	cmd.Flags().StringVarP(&codespaceName, "codespace", "c", "", "Codespace name to connect to")

	return cmd
}

func defaultTunnelDeps() tunnelDeps {
	return tunnelDeps{
		socketPath: client.UnixSocketPath,
		statusUnix: func(ctx context.Context, socketPath string) error {
			return checkStatus(ctx, client.NewWithSocketPath(socketPath))
		},
		startServer: startServerInBackground,
		listCodespaces: func(ctx context.Context) ([]codespace, error) {
			cmd := exec.CommandContext(ctx, "gh", "cs", "list", "--json", "name,state")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("list codespaces: %w: %s", err, strings.TrimSpace(string(output)))
			}

			var codespaces []codespace
			if err := json.Unmarshal(output, &codespaces); err != nil {
				return nil, fmt.Errorf("parse codespaces: %w", err)
			}
			return codespaces, nil
		},
		runTunnel: func(ctx context.Context, codespaceName, socketPath string) error {
			forward := fmt.Sprintf("localhost:%s:%s", rdmTunnelPort, socketPath)
			cmd := exec.CommandContext(ctx, "gh", "cs", "ssh", "-c", codespaceName, "--", "-o", "ExitOnForwardFailure=yes", "-N", "-R", forward)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("run codespaces tunnel: %w", err)
			}
			return nil
		},
	}
}

func runTunnel(ctx context.Context, out io.Writer, codespaceName string, deps tunnelDeps) error {
	socketPath := deps.socketPath()
	if err := ensureLocalServer(ctx, out, socketPath, deps); err != nil {
		return err
	}

	resolvedCodespace, err := resolveCodespace(ctx, codespaceName, deps.listCodespaces)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Starting tunnel to codespace %q\n", resolvedCodespace)
	fmt.Fprintf(out, "Forwarding localhost:%s to %s\n", rdmTunnelPort, socketPath)
	fmt.Fprintln(out, "Press Ctrl-C to stop the tunnel.")

	return deps.runTunnel(ctx, resolvedCodespace, socketPath)
}

func ensureLocalServer(ctx context.Context, out io.Writer, socketPath string, deps tunnelDeps) error {
	if err := deps.statusUnix(ctx, socketPath); err == nil {
		fmt.Fprintf(out, "✓ Local server is running at %s\n", socketPath)
		return nil
	}

	fmt.Fprintln(out, "Starting local gh-rdm server...")
	if err := deps.startServer(); err != nil {
		return fmt.Errorf("start local server: %w", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := deps.statusUnix(ctx, socketPath); err == nil {
			fmt.Fprintf(out, "✓ Local server started at %s\n", socketPath)
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("local server did not become ready: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}

	return fmt.Errorf("local server did not become ready: %w", lastErr)
}

func resolveCodespace(ctx context.Context, requested string, listCodespaces func(context.Context) ([]codespace, error)) (string, error) {
	if requested != "" {
		return requested, nil
	}

	codespaces, err := listCodespaces(ctx)
	if err != nil {
		return "", err
	}
	if len(codespaces) == 0 {
		return "", fmt.Errorf("no codespaces found; pass a codespace name with `gh rdm tunnel <codespace>`")
	}
	if len(codespaces) == 1 {
		return codespaces[0].Name, nil
	}

	var names []string
	for _, codespace := range codespaces {
		if codespace.State == "" {
			names = append(names, codespace.Name)
			continue
		}
		names = append(names, fmt.Sprintf("%s (%s)", codespace.Name, codespace.State))
	}
	return "", fmt.Errorf("multiple codespaces found; pass one with `gh rdm tunnel <codespace>`:\n  %s", strings.Join(names, "\n  "))
}

func startServerInBackground() error {
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

	cmd := exec.Command(os.Args[0], "server")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		return err
	}
	return nil
}
