package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxbeizer/gh-rdm/internal/client"
)

// mockRunner records calls and returns configured values.
type mockRunner struct {
	copiedText string
	pasteData  []byte
	pasteErr   error
	openedURL  string
	copyErr    error
	openErr    error
}

func (m *mockRunner) Copy(text string) error {
	m.copiedText = text
	return m.copyErr
}

func (m *mockRunner) Paste() ([]byte, error) {
	return m.pasteData, m.pasteErr
}

func (m *mockRunner) Open(target string) error {
	m.openedURL = target
	return m.openErr
}

func sendCommand(t *testing.T, srv http.Handler, cmd client.Command) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	return rec
}

func TestCopyCommand(t *testing.T) {
	mock := &mockRunner{}
	srv := New(mock, "/tmp/test.sock", log.Default())

	rec := sendCommand(t, srv, client.Command{Name: "copy", Arguments: []string{"hello"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if mock.copiedText != "hello" {
		t.Fatalf("expected copied text %q, got %q", "hello", mock.copiedText)
	}
}

func TestPasteCommand(t *testing.T) {
	mock := &mockRunner{pasteData: []byte("clipboard content")}
	srv := New(mock, "/tmp/test.sock", log.Default())

	rec := sendCommand(t, srv, client.Command{Name: "paste"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "clipboard content" {
		t.Fatalf("expected %q, got %q", "clipboard content", rec.Body.String())
	}
}

func TestOpenCommand(t *testing.T) {
	mock := &mockRunner{}
	srv := New(mock, "/tmp/test.sock", log.Default())

	rec := sendCommand(t, srv, client.Command{Name: "open", Arguments: []string{"https://example.com"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if mock.openedURL != "https://example.com" {
		t.Fatalf("expected opened URL %q, got %q", "https://example.com", mock.openedURL)
	}
}

func TestStatusCommand(t *testing.T) {
	mock := &mockRunner{}
	srv := New(mock, "/tmp/test.sock", log.Default())

	rec := sendCommand(t, srv, client.Command{Name: "status"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var status map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if status["status"] != "running" {
		t.Fatalf("expected status %q, got %q", "running", status["status"])
	}
}

func TestCopyCommandError(t *testing.T) {
	mock := &mockRunner{copyErr: fmt.Errorf("copy broke")}
	srv := New(mock, "/tmp/test.sock", log.Default())

	rec := sendCommand(t, srv, client.Command{Name: "copy", Arguments: []string{"hello"}})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
