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
	var copyRef bool

	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Fetch the latest screenshot from the local machine",
		Long: `Fetch the latest screenshot from the local machine's Desktop via
the gh-rdm tunnel and save it to a file on the remote machine.

Outputs the file path as an @ reference, ready to paste into Copilot CLI.
By default, the @ reference is also copied to your clipboard.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchImage(cmd, "screenshot", outputDir, copyRef)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "/tmp", "Directory to save the screenshot")
	cmd.Flags().BoolVarP(&copyRef, "copy", "c", true, "Copy the @ reference to clipboard")

	return cmd
}

func newClipboardImageCmd() *cobra.Command {
	var outputDir string
	var copyRef bool

	cmd := &cobra.Command{
		Use:   "clipboard-image",
		Short: "Fetch the clipboard image from the local machine",
		Long: `Grab the current image from the local machine's clipboard via
the gh-rdm tunnel and save it to a file on the remote machine.

Use ⌘⇧⌃4 to screenshot directly to clipboard, then run this command.
Outputs the file path as an @ reference, ready to paste into Copilot CLI.
By default, the @ reference is also copied to your clipboard.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchImage(cmd, "clipboard-image", outputDir, copyRef)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "/tmp", "Directory to save the image")
	cmd.Flags().BoolVarP(&copyRef, "copy", "c", true, "Copy the @ reference to clipboard")

	return cmd
}

func fetchImage(cmd *cobra.Command, commandName string, outputDir string, copyRef bool) error {
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

	ref := fmt.Sprintf("@%s", outPath)

	fmt.Fprintf(cmd.OutOrStdout(), "📸 Saved: %s (%d bytes)\n", outPath, len(data))
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", ref)

	if copyRef {
		c2 := client.New()
		if _, err := c2.SendCommand(cmd.Context(), "copy", ref); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠️  Could not copy to clipboard: %v\n", err)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "📋 Copied to clipboard\n")
		}
	}

	return nil
}
