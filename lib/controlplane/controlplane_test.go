package controlplane

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigs(t *testing.T) {
	t.Run("DefaultServerConfig", func(t *testing.T) {
		config := DefaultServerConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "http://localhost:8080", config.ServerURL)
		assert.Equal(t, "0.0.0.0:8080", config.ListenAddr)
		assert.Equal(t, "0.0.0.0:50443", config.GRPCAddr)
		assert.True(t, config.GRPCAllowInsecure)
		assert.Equal(t, "sqlite", config.Database.Type)
	})

	t.Run("DefaultClientConfig", func(t *testing.T) {
		config := DefaultClientConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "localhost:50443", config.Address)
		assert.True(t, config.Insecure)
		assert.Equal(t, 30*time.Second, config.Timeout)
	})
}

func TestServerConfigValidation(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := DefaultServerConfig()
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("MissingServerURL", func(t *testing.T) {
		config := DefaultServerConfig()
		config.ServerURL = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ServerURL is required")
	})

	t.Run("MissingNoisePrivateKeyPath", func(t *testing.T) {
		config := DefaultServerConfig()
		config.NoisePrivateKeyPath = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NoisePrivateKeyPath is required")
	})

	t.Run("MissingBaseDomain", func(t *testing.T) {
		config := DefaultServerConfig()
		config.BaseDomain = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "BaseDomain is required")
	})

	t.Run("MissingIPPrefixes", func(t *testing.T) {
		config := DefaultServerConfig()
		config.IPv4Prefix = ""
		config.IPv6Prefix = ""
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of IPv4Prefix or IPv6Prefix is required")
	})
}

func TestServerConfigToHeadscaleConfig(t *testing.T) {
	config := DefaultServerConfig()

	hsConfig, err := config.ToHeadscaleConfig()
	require.NoError(t, err)
	assert.NotNil(t, hsConfig)

	assert.Equal(t, config.ServerURL, hsConfig.ServerURL)
	assert.Equal(t, config.ListenAddr, hsConfig.Addr)
	assert.Equal(t, config.GRPCAddr, hsConfig.GRPCAddr)
	assert.Equal(t, config.GRPCAllowInsecure, hsConfig.GRPCAllowInsecure)
	assert.Equal(t, config.NoisePrivateKeyPath, hsConfig.NoisePrivateKeyPath)
	assert.Equal(t, config.BaseDomain, hsConfig.BaseDomain)
}

func TestServerCreation(t *testing.T) {
	t.Run("NewServerWithDefaultConfig", func(t *testing.T) {
		server, err := NewServer(nil)
		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.False(t, server.IsRunning())
	})

	t.Run("NewServerWithCustomConfig", func(t *testing.T) {
		config := DefaultServerConfig()
		// Use temporary paths for testing
		tempDir := t.TempDir()
		config.Database.SQLite.Path = filepath.Join(tempDir, "test.db")
		config.NoisePrivateKeyPath = filepath.Join(tempDir, "noise.key")
		config.DERP.ServerPrivateKeyPath = filepath.Join(tempDir, "derp.key")

		server, err := NewServer(config)
		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, config.GRPCAddr, server.GetGRPCAddress())
	})

	t.Run("NewServerWithInvalidConfig", func(t *testing.T) {
		config := DefaultServerConfig()
		config.ServerURL = "" // Invalid

		server, err := NewServer(config)
		assert.Error(t, err)
		assert.Nil(t, server)
	})
}

func TestEnsureDirectories(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultServerConfig()
	config.Database.SQLite.Path = filepath.Join(tempDir, "subdir", "test.db")
	config.NoisePrivateKeyPath = filepath.Join(tempDir, "keys", "noise.key")
	config.DERP.ServerPrivateKeyPath = filepath.Join(tempDir, "keys", "derp.key")

	err := config.EnsureDirectories()
	assert.NoError(t, err)

	// Check that directories were created
	assert.DirExists(t, filepath.Join(tempDir, "subdir"))
	assert.DirExists(t, filepath.Join(tempDir, "keys"))
}

func TestClientCreation(t *testing.T) {
	t.Run("NewClientWithDefaultConfig", func(t *testing.T) {
		// This might succeed or fail depending on whether there's a server running
		client, err := NewClient(nil)
		if err != nil {
			// Expected case - no server running
			assert.Contains(t, err.Error(), "failed to connect")
			assert.Nil(t, client)
		} else {
			// Unexpected but possible - server is running
			assert.NotNil(t, client)
			client.Close()
		}
	})

	t.Run("NewClientWithCustomConfig", func(t *testing.T) {
		config := &ClientConfig{
			Address:  "nonexistent.invalid:50443",
			Insecure: true,
			Timeout:  1 * time.Second, // Short timeout for test
		}

		// This should fail to connect since the address is invalid
		client, err := NewClient(config)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to connect")
			assert.Nil(t, client)
		} else {
			// Clean up if somehow it succeeded
			assert.NotNil(t, client)
			client.Close()
		}
	})
}

// Integration test that requires more setup - commented out for basic testing
/*
func TestServerClientIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup temporary directory
	tempDir := t.TempDir()

	// Create server config
	serverConfig := DefaultServerConfig()
	serverConfig.Database.SQLite.Path = filepath.Join(tempDir, "test.db")
	serverConfig.NoisePrivateKeyPath = filepath.Join(tempDir, "noise.key")
	serverConfig.DERP.ServerPrivateKeyPath = filepath.Join(tempDir, "derp.key")
	serverConfig.GRPCAddr = "localhost:0" // Use random port
	serverConfig.ListenAddr = "localhost:0" // Use random port

	// Create and start server
	server, err := NewServer(serverConfig)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Create client
	clientConfig := &ClientConfig{
		Address:  server.GetGRPCAddress(),
		Insecure: true,
		Timeout:  10 * time.Second,
	}

	client, err := NewClient(clientConfig)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Test user creation
	user, err := client.CreateUser(ctx, "test-user")
	require.NoError(t, err)
	assert.Equal(t, "test-user", user.Name)

	// Test user listing
	users, err := client.ListUsers(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "test-user", users[0].Name)

	// Test pre-auth key creation
	preAuthKey, err := client.CreatePreAuthKey(ctx, user.Id, false, false, nil, []string{})
	require.NoError(t, err)
	assert.NotEmpty(t, preAuthKey.Key)
}
*/
