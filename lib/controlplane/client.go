package controlplane

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// client implements the ControlPlaneClient interface
type client struct {
	conn   *grpc.ClientConn
	client v1.HeadscaleServiceClient
	config *ClientConfig
}

// NewClient creates a new control plane client with the given configuration
func NewClient(config *ClientConfig) (ControlPlaneClient, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	// Set up connection options
	var opts []grpc.DialOption

	if config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Connect to the server
	conn, err := grpc.DialContext(ctx, config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to headscale server at %s: %w", config.Address, err)
	}

	return &client{
		conn:   conn,
		client: v1.NewHeadscaleServiceClient(conn),
		config: config,
	}, nil
}

// Close closes the client connection
func (c *client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// getContext returns a context with API key if configured
func (c *client) getContext(ctx context.Context) context.Context {
	if c.config.APIKey != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.config.APIKey)
	}
	return ctx
}

// User Management

func (c *client) CreateUser(ctx context.Context, name string) (*v1.User, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.CreateUser(ctx, &v1.CreateUserRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return resp.User, nil
}

func (c *client) ListUsers(ctx context.Context) ([]*v1.User, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.ListUsers(ctx, &v1.ListUsersRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return resp.Users, nil
}

func (c *client) DeleteUser(ctx context.Context, userID uint64) error {
	ctx = c.getContext(ctx)
	_, err := c.client.DeleteUser(ctx, &v1.DeleteUserRequest{
		Id: userID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (c *client) RenameUser(ctx context.Context, userID uint64, newName string) (*v1.User, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.RenameUser(ctx, &v1.RenameUserRequest{
		OldId:   userID,
		NewName: newName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to rename user: %w", err)
	}
	return resp.User, nil
}

// Node Management

func (c *client) ListNodes(ctx context.Context, userID uint64) ([]*v1.Node, error) {
	ctx = c.getContext(ctx)

	// First get the user name from ID
	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var userName string
	for _, user := range users {
		if user.Id == userID {
			userName = user.Name
			break
		}
	}

	if userName == "" {
		return nil, fmt.Errorf("user with ID %d not found", userID)
	}

	resp, err := c.client.ListNodes(ctx, &v1.ListNodesRequest{
		User: userName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	return resp.Nodes, nil
}

func (c *client) GetNode(ctx context.Context, nodeID uint64) (*v1.Node, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.GetNode(ctx, &v1.GetNodeRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	return resp.Node, nil
}

func (c *client) DeleteNode(ctx context.Context, nodeID uint64) error {
	ctx = c.getContext(ctx)
	_, err := c.client.DeleteNode(ctx, &v1.DeleteNodeRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}
	return nil
}

func (c *client) ExpireNode(ctx context.Context, nodeID uint64) (*v1.Node, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.ExpireNode(ctx, &v1.ExpireNodeRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to expire node: %w", err)
	}
	return resp.Node, nil
}

func (c *client) RenameNode(ctx context.Context, nodeID uint64, newName string) (*v1.Node, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.RenameNode(ctx, &v1.RenameNodeRequest{
		NodeId:  nodeID,
		NewName: newName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to rename node: %w", err)
	}
	return resp.Node, nil
}

func (c *client) MoveNode(ctx context.Context, nodeID uint64, userID uint64) (*v1.Node, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.MoveNode(ctx, &v1.MoveNodeRequest{
		NodeId: nodeID,
		User:   userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to move node: %w", err)
	}
	return resp.Node, nil
}

func (c *client) RegisterNode(ctx context.Context, userID uint64, key string) (*v1.Node, error) {
	ctx = c.getContext(ctx)

	// First get the user name from ID
	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var userName string
	for _, user := range users {
		if user.Id == userID {
			userName = user.Name
			break
		}
	}

	if userName == "" {
		return nil, fmt.Errorf("user with ID %d not found", userID)
	}

	resp, err := c.client.RegisterNode(ctx, &v1.RegisterNodeRequest{
		User: userName,
		Key:  key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register node: %w", err)
	}
	return resp.Node, nil
}

// Pre-auth Key Management

func (c *client) CreatePreAuthKey(ctx context.Context, userID uint64, reusable bool, ephemeral bool, expiration *time.Time, aclTags []string) (*v1.PreAuthKey, error) {
	ctx = c.getContext(ctx)

	req := &v1.CreatePreAuthKeyRequest{
		User:      userID,
		Reusable:  reusable,
		Ephemeral: ephemeral,
		AclTags:   aclTags,
	}

	if expiration != nil {
		req.Expiration = timestamppb.New(*expiration)
	}

	resp, err := c.client.CreatePreAuthKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-auth key: %w", err)
	}
	return resp.PreAuthKey, nil
}

func (c *client) ListPreAuthKeys(ctx context.Context, userID uint64) ([]*v1.PreAuthKey, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.ListPreAuthKeys(ctx, &v1.ListPreAuthKeysRequest{
		User: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pre-auth keys: %w", err)
	}
	return resp.PreAuthKeys, nil
}

func (c *client) ExpirePreAuthKey(ctx context.Context, userID uint64, key string) error {
	ctx = c.getContext(ctx)
	_, err := c.client.ExpirePreAuthKey(ctx, &v1.ExpirePreAuthKeyRequest{
		User: userID,
		Key:  key,
	})
	if err != nil {
		return fmt.Errorf("failed to expire pre-auth key: %w", err)
	}
	return nil
}

// API Key Management

func (c *client) CreateAPIKey(ctx context.Context, expiration *time.Time) (string, error) {
	ctx = c.getContext(ctx)

	req := &v1.CreateApiKeyRequest{}
	if expiration != nil {
		req.Expiration = timestamppb.New(*expiration)
	}

	resp, err := c.client.CreateApiKey(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}
	return resp.ApiKey, nil
}

func (c *client) ListAPIKeys(ctx context.Context) ([]*v1.ApiKey, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.ListApiKeys(ctx, &v1.ListApiKeysRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	return resp.ApiKeys, nil
}

func (c *client) ExpireAPIKey(ctx context.Context, prefix string) error {
	ctx = c.getContext(ctx)
	_, err := c.client.ExpireApiKey(ctx, &v1.ExpireApiKeyRequest{
		Prefix: prefix,
	})
	if err != nil {
		return fmt.Errorf("failed to expire API key: %w", err)
	}
	return nil
}

func (c *client) DeleteAPIKey(ctx context.Context, prefix string) error {
	ctx = c.getContext(ctx)
	_, err := c.client.DeleteApiKey(ctx, &v1.DeleteApiKeyRequest{
		Prefix: prefix,
	})
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// Policy Management

func (c *client) GetPolicy(ctx context.Context) (string, error) {
	ctx = c.getContext(ctx)
	resp, err := c.client.GetPolicy(ctx, &v1.GetPolicyRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to get policy: %w", err)
	}
	return resp.Policy, nil
}

func (c *client) SetPolicy(ctx context.Context, policy string) error {
	ctx = c.getContext(ctx)
	_, err := c.client.SetPolicy(ctx, &v1.SetPolicyRequest{
		Policy: policy,
	})
	if err != nil {
		return fmt.Errorf("failed to set policy: %w", err)
	}
	return nil
}
