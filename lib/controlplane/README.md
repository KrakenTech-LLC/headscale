# Headscale Control Plane Library

This library provides a simplified interface for starting and controlling a Headscale control plane server programmatically. It's designed for distributed applications where you need to embed a Tailscale-compatible control plane server and manage it via gRPC.

## Features

- **Easy Server Management**: Start/stop a Headscale control plane server with minimal configuration
- **gRPC Client Interface**: Full-featured client for managing users, nodes, and policies
- **Simplified Configuration**: Sensible defaults with easy customization
- **Production Ready**: Built on top of the official Headscale implementation

## Quick Start

### 1. Start a Control Plane Server

```go
package main

import (
    "log"
    "github.com/juanfont/headscale/lib/controlplane"
)

func main() {
    // Create server with default configuration
    config := controlplane.DefaultServerConfig()
    config.ServerURL = "http://localhost:8080"
    config.Database.SQLite.Path = "/tmp/headscale.db"
    
    server, err := controlplane.NewServer(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start the server
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Control plane server running on %s", server.GetGRPCAddress())
    
    // Keep running...
    select {}
}
```

### 2. Connect and Manage via gRPC

```go
package main

import (
    "context"
    "log"
    "github.com/juanfont/headscale/lib/controlplane"
)

func main() {
    // Connect to the control plane
    clientConfig := controlplane.DefaultClientConfig()
    clientConfig.Address = "localhost:50443"
    
    client, err := controlplane.NewClient(clientConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    ctx := context.Background()
    
    // Create a user
    user, err := client.CreateUser(ctx, "my-user")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create a pre-auth key for device registration
    preAuthKey, err := client.CreatePreAuthKey(ctx, user.Id, false, false, nil, []string{})
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User: %s, PreAuth Key: %s", user.Name, preAuthKey.Key)
}
```

## Configuration

### Server Configuration

The `ServerConfig` struct provides comprehensive configuration options:

```go
config := &controlplane.ServerConfig{
    ServerURL:         "https://headscale.example.com",
    ListenAddr:        "0.0.0.0:8080",
    GRPCAddr:          "0.0.0.0:50443",
    GRPCAllowInsecure: false,
    
    Database: controlplane.DatabaseConfig{
        Type: "postgres", // or "sqlite"
        Postgres: controlplane.PostgresConfig{
            Host:     "localhost",
            Port:     5432,
            Database: "headscale",
            Username: "headscale",
            Password: "password",
        },
    },
    
    NoisePrivateKeyPath: "/etc/headscale/noise_private.key",
    BaseDomain:          "headscale.example.com",
    IPv4Prefix:          "100.64.0.0/10",
    IPv6Prefix:          "fd7a:115c:a1e0::/48",
    
    DERP: controlplane.DERPConfig{
        ServerEnabled: true,
        ServerRegionID: 999,
        ServerRegionCode: "headscale",
        ServerRegionName: "Headscale Embedded DERP",
    },
    
    TLS: controlplane.TLSConfig{
        LetsEncryptHostname: "headscale.example.com",
    },
}
```

### Client Configuration

```go
clientConfig := &controlplane.ClientConfig{
    Address:  "headscale.example.com:50443",
    APIKey:   "your-api-key", // Optional for authenticated access
    Insecure: false,
    Timeout:  30 * time.Second,
}
```

## API Reference

### Server Interface

```go
type ControlPlaneServer interface {
    Start() error
    Stop() error
    GetGRPCAddress() string
    IsRunning() bool
}
```

### Client Interface

The client provides methods for:

- **User Management**: `CreateUser`, `ListUsers`, `DeleteUser`, `RenameUser`
- **Node Management**: `ListNodes`, `GetNode`, `DeleteNode`, `ExpireNode`, `RenameNode`, `MoveNode`, `RegisterNode`
- **Pre-auth Keys**: `CreatePreAuthKey`, `ListPreAuthKeys`, `ExpirePreAuthKey`
- **API Keys**: `CreateAPIKey`, `ListAPIKeys`, `ExpireAPIKey`, `DeleteAPIKey`
- **Policy Management**: `GetPolicy`, `SetPolicy`

## Use Cases

### Distributed Application Control Plane

```go
// In your main application server (the "Helm")
server, _ := controlplane.NewServer(config)
server.Start()

// In your worker nodes
client, _ := controlplane.NewClient(clientConfig)
user, _ := client.CreateUser(ctx, "worker-node-1")
preAuthKey, _ := client.CreatePreAuthKey(ctx, user.Id, false, false, nil, []string{})

// Use the pre-auth key to connect the worker node to the tailnet
// tailscale up --login-server=https://your-helm-server --authkey=<preAuthKey>
```

### Dynamic Node Registration

```go
// Register a new client dynamically
func RegisterNewClient(client controlplane.ControlPlaneClient, clientName string) (string, error) {
    ctx := context.Background()
    
    // Create user for the client
    user, err := client.CreateUser(ctx, clientName)
    if err != nil {
        return "", err
    }
    
    // Create a single-use pre-auth key
    expiration := time.Now().Add(1 * time.Hour)
    preAuthKey, err := client.CreatePreAuthKey(ctx, user.Id, false, false, &expiration, []string{})
    if err != nil {
        return "", err
    }
    
    return preAuthKey.Key, nil
}
```

## Requirements

- Go 1.21+
- SQLite or PostgreSQL database
- Network access for DERP/STUN (if using embedded DERP server)

## Security Considerations

- Always use TLS in production (`GRPCAllowInsecure: false`)
- Secure your database connection
- Use API keys for client authentication
- Consider network policies for gRPC access
- Regularly rotate pre-auth keys and API keys

## Examples

See the `examples/` directory for complete working examples:

- `basic_usage.go`: Complete example showing server startup and client operations
- More examples coming soon...

## Contributing

This library is part of the Headscale project. Please refer to the main project for contribution guidelines.

## License

Same as Headscale project (BSD 3-Clause License).
