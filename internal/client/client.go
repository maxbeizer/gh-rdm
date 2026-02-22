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
	return NewWithSocketPath(UnixSocketPath())
}

func NewWithSocketPath(socketPath string) *Client {
	if os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return &Client{
			path:       "http://localhost:7391",
			httpClient: http.Client{Timeout: 10 * time.Second},
		}
	}

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

	return io.ReadAll(resp.Body)
}
