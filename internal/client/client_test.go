package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUnixSocketPath(t *testing.T) {
	tmp := strings.TrimRight(os.TempDir(), "/")
	expected := tmp + "/gh-rdm.sock"
	got := UnixSocketPath()
	if got != expected {
		t.Errorf("UnixSocketPath() = %q, want %q", got, expected)
	}
}

func TestSendCommand(t *testing.T) {
	var receivedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := &Client{
		path:       ts.URL,
		httpClient: *ts.Client(),
	}

	resp, err := c.SendCommand(context.Background(), "copy", "hello world")
	if err != nil {
		t.Fatalf("SendCommand() error: %v", err)
	}

	if string(resp) != "ok" {
		t.Errorf("SendCommand() response = %q, want %q", string(resp), "ok")
	}

	var cmd Command
	if err := json.Unmarshal(receivedBody, &cmd); err != nil {
		t.Fatalf("unmarshaling request body: %v", err)
	}

	if cmd.Name != "copy" {
		t.Errorf("command name = %q, want %q", cmd.Name, "copy")
	}

	if len(cmd.Arguments) != 1 || cmd.Arguments[0] != "hello world" {
		t.Errorf("command arguments = %v, want [\"hello world\"]", cmd.Arguments)
	}
}

func TestSendCommandHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "copy failed", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := &Client{
		path:       ts.URL,
		httpClient: *ts.Client(),
	}

	_, err := c.SendCommand(context.Background(), "copy", "hello world")
	if err == nil {
		t.Fatal("SendCommand() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Fatalf("SendCommand() error = %q, want status", err)
	}
}

func TestNewUsesTCPInSSHEnvironment(t *testing.T) {
	t.Setenv("SSH_TTY", "/dev/pts/1")

	c := New()

	if c.path != "http://localhost:7391" {
		t.Fatalf("New() path = %q, want tcp path", c.path)
	}
}

func TestNewUsesTCPInCodespaceEnvironment(t *testing.T) {
	t.Setenv("CODESPACES", "true")

	c := New()

	if c.path != "http://localhost:7391" {
		t.Fatalf("New() path = %q, want tcp path", c.path)
	}
}

func TestNewWithSocketPathIgnoresSSHEnvironment(t *testing.T) {
	t.Setenv("SSH_TTY", "/dev/pts/1")

	c := NewWithSocketPath("/tmp/example.sock")

	if c.path != "http://unix:///tmp/example.sock" {
		t.Fatalf("NewWithSocketPath() path = %q, want unix path", c.path)
	}
	if c.httpClient.Transport == nil {
		t.Fatal("NewWithSocketPath() transport = nil, want unix transport")
	}
}
