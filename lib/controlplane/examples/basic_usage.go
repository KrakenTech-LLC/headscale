package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/juanfont/headscale/lib/controlplane"
)

func main() {
	// Example 1: Start a control plane server
	fmt.Println("=== Starting Control Plane Server ===")

	// Create server configuration
	config := controlplane.DefaultServerConfig()
	config.ServerURL = "http://localhost:8080"
	config.GRPCAddr = "localhost:50443"
	config.ListenAddr = "localhost:8080"
	config.Database.SQLite.Path = "/tmp/headscale_example.db"
	config.NoisePrivateKeyPath = "/tmp/headscale_noise_example.key"
	config.DERP.ServerPrivateKeyPath = "/tmp/headscale_derp_example.key"

	// Create and start the server
	server, err := controlplane.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Printf("Server started on gRPC: %s\n", server.GetGRPCAddress())

	// Wait a moment for the server to be ready
	time.Sleep(2 * time.Second)

	// Example 2: Connect a client and manage users/nodes
	fmt.Println("\n=== Managing Users and Nodes ===")

	// Create client configuration
	clientConfig := controlplane.DefaultClientConfig()
	clientConfig.Address = server.GetGRPCAddress()

	// Create client
	client, err := controlplane.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a user
	user, err := client.CreateUser(ctx, "example-user")
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.Id)

	// List users
	users, err := client.ListUsers(ctx)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("Total users: %d\n", len(users))
	for _, u := range users {
		fmt.Printf("  - %s (ID: %d)\n", u.Name, u.Id)
	}

	// Create a pre-auth key for the user
	expiration := time.Now().Add(24 * time.Hour)
	preAuthKey, err := client.CreatePreAuthKey(ctx, user.Id, false, false, &expiration, []string{})
	if err != nil {
		log.Fatalf("Failed to create pre-auth key: %v", err)
	}
	fmt.Printf("Created pre-auth key: %s\n", preAuthKey.Key)

	// List pre-auth keys
	keys, err := client.ListPreAuthKeys(ctx, user.Id)
	if err != nil {
		log.Fatalf("Failed to list pre-auth keys: %v", err)
	}
	fmt.Printf("Pre-auth keys for user %s: %d\n", user.Name, len(keys))
	for _, key := range keys {
		fmt.Printf("  - %s (Used: %t, Reusable: %t)\n", key.Key, key.Used, key.Reusable)
	}

	// Create an API key
	apiKeyExpiration := time.Now().Add(7 * 24 * time.Hour) // 7 days
	apiKey, err := client.CreateAPIKey(ctx, &apiKeyExpiration)
	if err != nil {
		log.Fatalf("Failed to create API key: %v", err)
	}
	fmt.Printf("Created API key: %s\n", apiKey)

	// List API keys
	apiKeys, err := client.ListAPIKeys(ctx)
	if err != nil {
		log.Fatalf("Failed to list API keys: %v", err)
	}
	fmt.Printf("Total API keys: %d\n", len(apiKeys))

	// Example 3: Policy management
	fmt.Println("\n=== Policy Management ===")

	// Set a basic policy
	basicPolicy := `{
		"hosts": {
			"example-host": "100.64.0.1"
		},
		"acls": [
			{
				"action": "accept",
				"src": ["*"],
				"dst": ["*:*"]
			}
		]
	}`

	if err := client.SetPolicy(ctx, basicPolicy); err != nil {
		log.Printf("Failed to set policy (this might fail if policy mode is not set to database): %v", err)
	} else {
		fmt.Println("Policy set successfully")
	}

	// Get current policy
	policy, err := client.GetPolicy(ctx)
	if err != nil {
		log.Printf("Failed to get policy: %v", err)
	} else {
		fmt.Printf("Current policy length: %d characters\n", len(policy))
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("The headscale control plane server is now running.")
	fmt.Println("You can:")
	fmt.Printf("  - Connect tailscale clients using: tailscale up --login-server=%s --authkey=%s\n", config.ServerURL, preAuthKey.Key)
	fmt.Printf("  - Access the web UI at: %s\n", config.ServerURL)
	fmt.Printf("  - Use the gRPC API at: %s\n", server.GetGRPCAddress())
	fmt.Println("\nPress Ctrl+C to stop the server...")

	// Keep the server running
	select {}
}
