package hostservice

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLatestScreenshot(t *testing.T) {
	dir := t.TempDir()

	// Create some fake screenshot files with different timestamps
	files := []struct {
		name    string
		content string
		delay   time.Duration
	}{
		{"Screenshot 2026-03-06 at 10.00.00.png", "older", 0},
		{"Screenshot 2026-03-06 at 12.00.00.png", "newest", 50 * time.Millisecond},
		{"not-a-screenshot.png", "ignore me", 100 * time.Millisecond},
	}

	for _, f := range files {
		time.Sleep(f.delay)
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte(f.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	svc := New()
	data, name, err := svc.LatestScreenshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "Screenshot 2026-03-06 at 12.00.00.png" {
		t.Fatalf("expected latest screenshot, got %q", name)
	}
	if string(data) != "newest" {
		t.Fatalf("expected content %q, got %q", "newest", string(data))
	}
}

func TestLatestScreenshotNoFiles(t *testing.T) {
	dir := t.TempDir()

	svc := New()
	_, _, err := svc.LatestScreenshot(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestLatestScreenshotDefaultDir(t *testing.T) {
	// When dir is empty, it should use ~/Desktop — just verify it doesn't panic
	svc := New()
	// This may return an error if ~/Desktop has no screenshots, but shouldn't panic
	_, _, _ = svc.LatestScreenshot("")
}
