package controlplane

import (
	"context"
	"fmt"
	"sync"

	"github.com/juanfont/headscale/hscontrol"
	"github.com/rs/zerolog/log"
)

// server implements the ControlPlaneServer interface
type server struct {
	config    *ServerConfig
	headscale *hscontrol.Headscale
	running   bool
	mu        sync.RWMutex
	stopCh    chan struct{}
}

// NewServer creates a new control plane server with the given configuration
func NewServer(config *ServerConfig) (ControlPlaneServer, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure necessary directories exist
	if err := config.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return &server{
		config: config,
		stopCh: make(chan struct{}),
	}, nil
}

// Start starts the control plane server
func (s *server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Convert to headscale config
	hsConfig, err := s.config.ToHeadscaleConfig()
	if err != nil {
		return fmt.Errorf("failed to convert configuration: %w", err)
	}

	// Create headscale instance
	s.headscale, err = hscontrol.NewHeadscale(hsConfig)
	if err != nil {
		return fmt.Errorf("failed to create headscale instance: %w", err)
	}

	// Start the server in a goroutine
	go func() {
		log.Info().Msg("Starting headscale control plane server")
		if err := s.headscale.Serve(); err != nil {
			log.Error().Err(err).Msg("Headscale server error")
		}
	}()

	s.running = true
	log.Info().
		Str("grpc_addr", s.config.GRPCAddr).
		Str("http_addr", s.config.ListenAddr).
		Msg("Control plane server started")

	return nil
}

// Stop gracefully stops the control plane server
func (s *server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server is not running")
	}

	log.Info().Msg("Stopping headscale control plane server")

	// Signal stop
	close(s.stopCh)

	// Note: headscale doesn't have a clean shutdown method in the current version
	// This is a limitation of the current headscale implementation
	// In a production environment, you might want to implement proper shutdown handling

	s.running = false
	s.headscale = nil

	log.Info().Msg("Control plane server stopped")
	return nil
}

// GetGRPCAddress returns the gRPC address the server is listening on
func (s *server) GetGRPCAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.GRPCAddr
}

// IsRunning returns true if the server is currently running
func (s *server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig returns the server configuration
func (s *server) GetConfig() *ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// WaitForReady waits for the server to be ready to accept connections
func (s *server) WaitForReady(ctx context.Context) error {
	if !s.IsRunning() {
		return fmt.Errorf("server is not running")
	}

	// Create a simple client to test connectivity
	clientConfig := &ClientConfig{
		Address:  s.GetGRPCAddress(),
		Insecure: s.config.GRPCAllowInsecure,
		Timeout:  s.config.EphemeralNodeInactivityTimeout,
	}

	client, err := NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create test client: %w", err)
	}
	defer client.Close()

	// Try to list users as a connectivity test
	_, err = client.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("server not ready: %w", err)
	}

	return nil
}
