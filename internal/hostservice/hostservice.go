package hostservice

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Runner defines clipboard and open operations on the host.
type Runner interface {
	Copy(text string) error
	Paste() ([]byte, error)
	Open(target string) error
}

// Service implements Runner using platform-native commands.
type Service struct{}

// New returns a new Service.
func New() *Service {
	return &Service{}
}

func (s *Service) Copy(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	return nil
}

func (s *Service) Paste() ([]byte, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("paste failed: %w", err)
	}

	return out, nil
}

func (s *Service) Open(target string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open failed: %w", err)
	}

	return nil
}
