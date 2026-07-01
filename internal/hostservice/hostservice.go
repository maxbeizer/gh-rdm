package hostservice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Runner defines clipboard and open operations on the host.
type Runner interface {
	Copy(text string) error
	Paste() ([]byte, error)
	Open(target string) error
	LatestScreenshot(dir string) ([]byte, string, error)
	ClipboardImage() ([]byte, error)
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

func (s *Service) LatestScreenshot(dir string) ([]byte, string, error) {
	if runtime.GOOS != "darwin" {
		return nil, "", fmt.Errorf("screenshot capture only supported on macOS")
	}

	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("get home dir: %w", err)
		}
		dir = filepath.Join(home, "Desktop")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("read screenshot dir: %w", err)
	}

	type fileWithTime struct {
		name    string
		modTime int64
	}

	var screenshots []fileWithTime
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "screenshot") && strings.HasSuffix(lower, ".png") {
			info, err := e.Info()
			if err != nil {
				continue
			}
			screenshots = append(screenshots, fileWithTime{name: name, modTime: info.ModTime().UnixNano()})
		}
	}

	if len(screenshots) == 0 {
		return nil, "", fmt.Errorf("no screenshots found in %s", dir)
	}

	sort.Slice(screenshots, func(i, j int) bool {
		return screenshots[i].modTime > screenshots[j].modTime
	})

	latest := screenshots[0].name
	data, err := os.ReadFile(filepath.Join(dir, latest))
	if err != nil {
		return nil, "", fmt.Errorf("read screenshot: %w", err)
	}

	return data, latest, nil
}

func (s *Service) ClipboardImage() ([]byte, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("clipboard image capture only supported on macOS")
	}

	tmpFile, err := os.CreateTemp("", "gh-rdm-clipboard-*.png")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	script := fmt.Sprintf(`try
	set theImage to the clipboard as «class PNGf»
	set theFile to open for access POSIX file "%s" with write permission
	write theImage to theFile
	close access theFile
	return "ok"
on error
	return "no image"
end try`, tmpPath)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("osascript failed: %w", err)
	}

	result := strings.TrimSpace(string(out))
	if result != "ok" {
		return nil, fmt.Errorf("no image on clipboard")
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("read clipboard image: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("clipboard image is empty")
	}

	return data, nil
}
