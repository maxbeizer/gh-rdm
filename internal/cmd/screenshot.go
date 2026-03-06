package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/spf13/cobra"
)

type imageResponse struct {
	Filename string `json:"filename"`
	Data     string `json:"data"`
}

func newScreenshotCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Fetch the latest screenshot from the local machine",
		Long: `Fetch the latest screenshot from the local machine's Desktop via
the gh-rdm tunnel and save it to a file on the remote machine.

Outputs the file path as an @ reference, ready to paste into Copilot CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchImage(cmd, "screenshot", outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "/tmp", "Directory to save the screenshot")

	return cmd
}

func newClipboardImageCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "clipboard-image",
		Short: "Fetch the clipboard image from the local machine",
		Long: `Grab the current image from the local machine's clipboard via
the gh-rdm tunnel and save it to a file on the remote machine.

Use ⌘⇧⌃4 to screenshot directly to clipboard, then run this command.
Outputs the file path as an @ reference, ready to paste into Copilot CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchImage(cmd, "clipboard-image", outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "/tmp", "Directory to save the image")

	return cmd
}

func fetchImage(cmd *cobra.Command, commandName string, outputDir string) error {
	c := client.New()
	result, err := c.SendCommand(cmd.Context(), commandName)
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}

	var resp imageResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w\nraw: %s", err, string(result))
	}

	data, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		return fmt.Errorf("failed to decode image data: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("screenshot-%s.png", timestamp)
	outPath := filepath.Join(outputDir, filename)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "📸 Saved: %s (%d bytes)\n", outPath, len(data))
	fmt.Fprintf(cmd.OutOrStdout(), "@ %s\n", outPath)

	return nil
}
