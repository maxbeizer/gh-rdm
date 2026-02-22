package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard for gh-rdm",
		RunE: func(cmd *cobra.Command, args []string) error {
			scanner := bufio.NewScanner(os.Stdin)
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// Step 1: Server Status
			fmt.Fprintln(out, "=== Step 1: Server Status ===")
			c := client.New()
			_, err := c.SendCommand(ctx, "status")
			if err == nil {
				fmt.Fprintln(out, "✓ Server is already running")
			} else {
				if askYesNo(scanner, "Start the server now? [Y/n]") {
					socketPath := client.UnixSocketPath()
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

					srvCmd := exec.CommandContext(ctx, os.Args[0], "server")
					srvCmd.Stdout = logFile
					srvCmd.Stderr = logFile
					if err := srvCmd.Start(); err != nil {
						logFile.Close()
						return fmt.Errorf("failed to start server: %w", err)
					}
					logFile.Close()

					fmt.Fprintf(out, "✓ Server started (socket: %s)\n", socketPath)
				} else {
					fmt.Fprintln(out, "Skipping. Run `gh rdm server` when ready.")
				}
			}

			// Step 2: SSH Config
			fmt.Fprintln(out, "\n=== Step 2: SSH Config ===")
			if askYesNo(scanner, "Configure SSH forwarding for a host? [Y/n]") {
				fmt.Fprint(out, "Enter the SSH host name (e.g., devbox, codespace): ")
				scanner.Scan()
				hostName := strings.TrimSpace(scanner.Text())
				if hostName != "" {
					if err := configureSSH(out, hostName); err != nil {
						fmt.Fprintf(out, "Warning: %v\n", err)
					}
				}
			}

			// Step 3: Integrations
			fmt.Fprintln(out, "\n=== Step 3: Integrations ===")
			options := []string{
				"Neovim clipboard",
				"GitHub CLI browser (gh config set browser)",
				"Shell aliases (pbcopy/open)",
				"All of the above",
				"None",
			}
			choice := askChoice(scanner, "Which integrations would you like to configure?", options)

			switch choice {
			case 1:
				printNeovimConfig(out)
			case 2:
				configureGHBrowser(out)
			case 3:
				printShellAliases(out)
			case 4:
				printNeovimConfig(out)
				configureGHBrowser(out)
				printShellAliases(out)
			case 5:
				fmt.Fprintln(out, "Skipping integrations.")
			}

			// Step 4: Done
			fmt.Fprintln(out, "\nSetup complete! Quick test:")
			fmt.Fprintln(out, "  echo \"hello\" | gh rdm copy && gh rdm paste")

			return nil
		},
	}
}

func askYesNo(scanner *bufio.Scanner, prompt string) bool {
	fmt.Print(prompt + " ")
	scanner.Scan()
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "" || answer == "y" || answer == "yes"
}

func askChoice(scanner *bufio.Scanner, prompt string, options []string) int {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Print("Enter choice: ")
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(options) {
		return len(options) // default to last option (None)
	}
	return choice
}

func configureSSH(out io.Writer, hostName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	socketPath := client.UnixSocketPath()
	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	remoteForwardLine := fmt.Sprintf("    RemoteForward 127.0.0.1:7391 %s", socketPath)

	data, err := os.ReadFile(sshConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new config
			if err := os.MkdirAll(filepath.Join(homeDir, ".ssh"), 0o700); err != nil {
				return err
			}
			content := fmt.Sprintf("\nHost %s\n%s\n", hostName, remoteForwardLine)
			if err := os.WriteFile(sshConfigPath, []byte(content), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(out, "✓ Added RemoteForward to ~/.ssh/config for host '%s'\n", hostName)
			return nil
		}
		return err
	}

	content := string(data)
	hostPattern := regexp.MustCompile(`(?im)^Host\s+` + regexp.QuoteMeta(hostName) + `\s*$`)
	if hostPattern.MatchString(content) {
		// Host block exists
		rfPattern := regexp.MustCompile(`(?i)RemoteForward.*gh-rdm`)
		if rfPattern.MatchString(content) {
			fmt.Fprintf(out, "✓ SSH config for host '%s' already has RemoteForward for gh-rdm\n", hostName)
			return nil
		}
		fmt.Fprintf(out, "⚠ Host '%s' exists in ~/.ssh/config but has no RemoteForward for gh-rdm.\n", hostName)
		fmt.Fprintf(out, "  Add this line to the Host %s block:\n", hostName)
		fmt.Fprintf(out, "  %s\n", remoteForwardLine)
		return nil
	}

	// Append new host block
	f, err := os.OpenFile(sshConfigPath, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\nHost %s\n%s\n", hostName, remoteForwardLine)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "✓ Added RemoteForward to ~/.ssh/config for host '%s'\n", hostName)
	return nil
}

func printNeovimConfig(out io.Writer) {
	fmt.Fprintln(out, "\n-- Add to your init.lua:")
	fmt.Fprintln(out, `vim.g.clipboard = {
  name = "gh-rdm",
  copy = { ["+"] = "gh rdm copy", ["*"] = "gh rdm copy" },
  paste = { ["+"] = "gh rdm paste", ["*"] = "gh rdm paste" },
  cache_enabled = true,
}`)
}

func configureGHBrowser(out io.Writer) {
	ghCmd := exec.Command("gh", "config", "set", "browser", "gh rdm open")
	output, err := ghCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(out, "⚠ Failed to set gh browser: %v\n%s\n", err, output)
		return
	}
	fmt.Fprintln(out, "✓ Set gh browser to \"gh rdm open\"")
}

func printShellAliases(out io.Writer) {
	fmt.Fprintln(out, "\n# Add to your shell profile (.bashrc, .zshrc, etc.):")
	fmt.Fprintln(out, `alias pbcopy="gh rdm copy"
alias pbpaste="gh rdm paste"
alias open="gh rdm open"
alias xdg-open="gh rdm open"`)
}
