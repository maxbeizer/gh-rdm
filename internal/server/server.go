package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/maxbeizer/gh-rdm/internal/client"
	"github.com/maxbeizer/gh-rdm/internal/hostservice"
)

// Server handles host-service commands over a unix socket.
type Server struct {
	host       hostservice.Runner
	path       string
	logger     *log.Logger
	httpServer *http.Server
	cancel     context.CancelFunc
}

// New creates a Server with sensible defaults.
func New(service hostservice.Runner, path string, logger *log.Logger) *Server {
	s := &Server{
		host:   service,
		path:   path,
		logger: logger,
	}

	s.httpServer = &http.Server{
		Handler:      s,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// ServeHTTP dispatches incoming commands.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}

	var cmd client.Command
	if err := json.Unmarshal(body, &cmd); err != nil {
		http.Error(w, fmt.Sprintf("parse command: %v", err), http.StatusBadRequest)
		return
	}

	switch cmd.Name {
	case "status":
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "running"}`)

	case "copy":
		if len(cmd.Arguments) < 1 {
			http.Error(w, "copy requires an argument", http.StatusBadRequest)
			return
		}
		if err := s.host.Copy(cmd.Arguments[0]); err != nil {
			http.Error(w, fmt.Sprintf("copy failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "paste":
		data, err := s.host.Paste()
		if err != nil {
			http.Error(w, fmt.Sprintf("paste failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write(data)

	case "open":
		if len(cmd.Arguments) < 1 {
			http.Error(w, "open requires an argument", http.StatusBadRequest)
			return
		}
		if err := s.host.Open(cmd.Arguments[0]); err != nil {
			http.Error(w, fmt.Sprintf("open failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "stop":
		if s.cancel != nil {
			s.cancel()
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, fmt.Sprintf("unknown command: %s", cmd.Name), http.StatusBadRequest)
	}
}

// Listen creates the unix socket and starts serving.
func (s *Server) Listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	ln, err := net.Listen("unix", s.path)
	if err != nil {
		if isAddrInUse(err) {
			// Check if existing socket is alive.
			c := client.NewWithSocketPath(s.path)
			if _, statusErr := c.SendCommand(ctx, "status"); statusErr == nil {
				cancel()
				return fmt.Errorf("server already running at %s", s.path)
			}
			// Stale socket — remove and retry.
			s.logger.Printf("removing stale socket %s", s.path)
			if removeErr := os.Remove(s.path); removeErr != nil {
				cancel()
				return fmt.Errorf("remove stale socket: %w", removeErr)
			}
			ln, err = net.Listen("unix", s.path)
			if err != nil {
				cancel()
				return fmt.Errorf("listen after cleanup: %w", err)
			}
		} else {
			cancel()
			return fmt.Errorf("listen: %w", err)
		}
	}

	return s.Serve(ctx, ln)
}

// Serve starts the HTTP server and blocks until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	s.logger.Printf("server listening on %s", s.path)

	select {
	case <-ctx.Done():
		s.logger.Println("shutting down server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *os.SyscallError
		if errors.As(opErr.Err, &sysErr) {
			return errors.Is(sysErr.Err, syscall.EADDRINUSE)
		}
	}
	return false
}
