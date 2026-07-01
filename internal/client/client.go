package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	RunLocal  = "unix"
	RunRemote = "tcp"
)

type Command struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
}

type Client struct {
	path       string
	httpClient http.Client
}

func UnixSocketPath() string {
	tmp := strings.TrimRight(os.TempDir(), "/")
	return tmp + "/gh-rdm.sock"
}

func New() *Client {
	if isRemoteEnvironment() {
		return NewWithTCPAddress("localhost:7391")
	}

	return NewWithSocketPath(UnixSocketPath())
}

func NewWithSocketPath(socketPath string) *Client {
	return &Client{
		path: "http://unix://" + socketPath,
		httpClient: http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

func NewWithTCPAddress(address string) *Client {
	return &Client{
		path:       "http://" + address,
		httpClient: http.Client{Timeout: 10 * time.Second},
	}
}

func isRemoteEnvironment() bool {
	return os.Getenv("SSH_TTY") != "" ||
		os.Getenv("SSH_CLIENT") != "" ||
		os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("CODESPACES") == "true" ||
		os.Getenv("CODESPACE_NAME") != ""
}

func (c *Client) SendCommand(ctx context.Context, commandName string, arguments ...string) ([]byte, error) {
	cmd := Command{
		Name:      commandName,
		Arguments: arguments,
	}

	body, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshaling command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending command: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	return responseBody, nil
}
