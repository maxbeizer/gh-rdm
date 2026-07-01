package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunDoctorLocalHealthy(t *testing.T) {
	var out bytes.Buffer
	deps := fakeDoctorDeps()

	err := runDoctor(context.Background(), &out, deps)
	if err != nil {
		t.Fatalf("runDoctor() error = %v, want nil", err)
	}

	output := out.String()
	if !strings.Contains(output, "✓ socket path exists: /tmp/gh-rdm.sock") {
		t.Fatalf("runDoctor() output missing socket success:\n%s", output)
	}
	if !strings.Contains(output, "skipped (not running in SSH or Codespaces environment)") {
		t.Fatalf("runDoctor() output missing remote skip:\n%s", output)
	}
}

func TestRunDoctorCodespaceBrokenTunnelPrintsRepairCommand(t *testing.T) {
	var out bytes.Buffer
	deps := fakeDoctorDeps()
	deps.getenv = func(key string) string {
		switch key {
		case "CODESPACES":
			return "true"
		case "CODESPACE_NAME":
			return "shiny-space"
		default:
			return ""
		}
	}
	deps.statusTCP = func(context.Context, string) error {
		return errors.New("connection refused")
	}

	err := runDoctor(context.Background(), &out, deps)
	if err == nil {
		t.Fatal("runDoctor() error = nil, want error")
	}

	output := out.String()
	for _, address := range []string{"localhost:7391", "127.0.0.1:7391", "[::1]:7391"} {
		if !strings.Contains(output, address) {
			t.Fatalf("runDoctor() output missing %s:\n%s", address, output)
		}
	}
	want := "gh cs ssh -c shiny-space -- -o ExitOnForwardFailure=yes -N -R localhost:7391:$(gh rdm socket)"
	if !strings.Contains(output, want) {
		t.Fatalf("runDoctor() output missing repair command %q:\n%s", want, output)
	}
}

func TestRunDoctorSSHHealthyTunnel(t *testing.T) {
	var out bytes.Buffer
	deps := fakeDoctorDeps()
	deps.statSocket = func(string) error {
		return errors.New("socket not on remote machine")
	}
	deps.statusUnix = func(context.Context, string) error {
		return errors.New("server not on remote machine")
	}
	deps.getenv = func(key string) string {
		if key == "SSH_CONNECTION" {
			return "remote"
		}
		return ""
	}

	err := runDoctor(context.Background(), &out, deps)
	if err != nil {
		t.Fatalf("runDoctor() error = %v, want nil", err)
	}

	output := out.String()
	if !strings.Contains(output, "checking TCP tunnel instead") {
		t.Fatalf("runDoctor() output missing unix skip:\n%s", output)
	}
	if strings.Contains(output, "Repair command") {
		t.Fatalf("runDoctor() output printed unexpected repair command:\n%s", output)
	}
}

func fakeDoctorDeps() doctorDeps {
	return doctorDeps{
		socketPath: func() string {
			return "/tmp/gh-rdm.sock"
		},
		statSocket: func(string) error {
			return nil
		},
		statusUnix: func(context.Context, string) error {
			return nil
		},
		statusTCP: func(context.Context, string) error {
			return nil
		},
		getenv: func(string) string {
			return ""
		},
	}
}
