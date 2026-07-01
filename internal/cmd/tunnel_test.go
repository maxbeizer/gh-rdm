package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunTunnelUsesRequestedCodespaceAndExistingServer(t *testing.T) {
	var out bytes.Buffer
	var ranCodespace, ranSocket string
	startedServer := false
	deps := fakeTunnelDeps()
	deps.startServer = func() error {
		startedServer = true
		return nil
	}
	deps.runTunnel = func(_ context.Context, codespaceName, socketPath string) error {
		ranCodespace = codespaceName
		ranSocket = socketPath
		return nil
	}

	err := runTunnel(context.Background(), &out, "my-space", deps)
	if err != nil {
		t.Fatalf("runTunnel() error = %v, want nil", err)
	}
	if startedServer {
		t.Fatal("runTunnel() started server, want existing server reuse")
	}
	if ranCodespace != "my-space" {
		t.Fatalf("runTunnel() codespace = %q, want %q", ranCodespace, "my-space")
	}
	if ranSocket != "/tmp/gh-rdm.sock" {
		t.Fatalf("runTunnel() socket = %q, want %q", ranSocket, "/tmp/gh-rdm.sock")
	}
}

func TestRunTunnelStartsServerAndSelectsOnlyCodespace(t *testing.T) {
	var out bytes.Buffer
	statusChecks := 0
	startedServer := false
	deps := fakeTunnelDeps()
	deps.statusUnix = func(context.Context, string) error {
		statusChecks++
		if startedServer {
			return nil
		}
		return errors.New("connection refused")
	}
	deps.startServer = func() error {
		startedServer = true
		return nil
	}
	deps.listCodespaces = func(context.Context) ([]codespace, error) {
		return []codespace{{Name: "only-space", State: "Available"}}, nil
	}

	err := runTunnel(context.Background(), &out, "", deps)
	if err != nil {
		t.Fatalf("runTunnel() error = %v, want nil", err)
	}
	if !startedServer {
		t.Fatal("runTunnel() did not start server")
	}
	if statusChecks < 2 {
		t.Fatalf("status checks = %d, want at least 2", statusChecks)
	}
	output := out.String()
	if !strings.Contains(output, "Local server started") {
		t.Fatalf("runTunnel() output missing start message:\n%s", output)
	}
	if !strings.Contains(output, "only-space") {
		t.Fatalf("runTunnel() output missing selected codespace:\n%s", output)
	}
}

func TestResolveCodespaceRequiresExplicitNameWhenMultipleExist(t *testing.T) {
	_, err := resolveCodespace(context.Background(), "", func(context.Context) ([]codespace, error) {
		return []codespace{
			{Name: "first", State: "Available"},
			{Name: "second", State: "Shutdown"},
		}, nil
	})
	if err == nil {
		t.Fatal("resolveCodespace() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "first (Available)") || !strings.Contains(err.Error(), "second (Shutdown)") {
		t.Fatalf("resolveCodespace() error missing codespace names: %v", err)
	}
}

func fakeTunnelDeps() tunnelDeps {
	return tunnelDeps{
		socketPath: func() string {
			return "/tmp/gh-rdm.sock"
		},
		statusUnix: func(context.Context, string) error {
			return nil
		},
		startServer: func() error {
			return nil
		},
		listCodespaces: func(context.Context) ([]codespace, error) {
			return nil, nil
		},
		runTunnel: func(context.Context, string, string) error {
			return nil
		},
	}
}
