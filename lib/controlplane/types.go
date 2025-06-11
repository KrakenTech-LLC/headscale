package controlplane

import (
	"context"
	"time"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
)

// ControlPlaneServer represents a headscale control plane server instance
type ControlPlaneServer interface {
	// Start starts the control plane server
	Start() error

	// Stop gracefully stops the control plane server
	Stop() error

	// GetGRPCAddress returns the gRPC address the server is listening on
	GetGRPCAddress() string

	// IsRunning returns true if the server is currently running
	IsRunning() bool
}

// ControlPlaneClient provides a high-level interface for managing the control plane
type ControlPlaneClient interface {
	// User Management
	CreateUser(ctx context.Context, name string) (*v1.User, error)
	ListUsers(ctx context.Context) ([]*v1.User, error)
	DeleteUser(ctx context.Context, userID uint64) error
	RenameUser(ctx context.Context, userID uint64, newName string) (*v1.User, error)

	// Node Management
	ListNodes(ctx context.Context, userID uint64) ([]*v1.Node, error)
	GetNode(ctx context.Context, nodeID uint64) (*v1.Node, error)
	DeleteNode(ctx context.Context, nodeID uint64) error
	ExpireNode(ctx context.Context, nodeID uint64) (*v1.Node, error)
	RenameNode(ctx context.Context, nodeID uint64, newName string) (*v1.Node, error)
	MoveNode(ctx context.Context, nodeID uint64, userID uint64) (*v1.Node, error)
	RegisterNode(ctx context.Context, userID uint64, key string) (*v1.Node, error)

	// Pre-auth Key Management
	CreatePreAuthKey(ctx context.Context, userID uint64, reusable bool, ephemeral bool, expiration *time.Time, aclTags []string) (*v1.PreAuthKey, error)
	ListPreAuthKeys(ctx context.Context, userID uint64) ([]*v1.PreAuthKey, error)
	ExpirePreAuthKey(ctx context.Context, userID uint64, key string) error

	// API Key Management
	CreateAPIKey(ctx context.Context, expiration *time.Time) (string, error)
	ListAPIKeys(ctx context.Context) ([]*v1.ApiKey, error)
	ExpireAPIKey(ctx context.Context, prefix string) error
	DeleteAPIKey(ctx context.Context, prefix string) error

	// Policy Management
	GetPolicy(ctx context.Context) (string, error)
	SetPolicy(ctx context.Context, policy string) error

	// Connection Management
	Close() error
}

// ServerConfig contains the configuration needed to start a headscale control plane server
type ServerConfig struct {
	// ServerURL is the public URL of the headscale server (e.g., "https://headscale.example.com")
	ServerURL string

	// ListenAddr is the address to listen on for HTTP traffic (default: "0.0.0.0:8080")
	ListenAddr string

	// GRPCAddr is the address to listen on for gRPC traffic (default: "0.0.0.0:50443")
	GRPCAddr string

	// GRPCAllowInsecure allows insecure gRPC connections (default: false)
	GRPCAllowInsecure bool

	// DatabaseConfig specifies the database configuration
	Database DatabaseConfig

	// NoisePrivateKeyPath is the path to the Noise protocol private key file
	NoisePrivateKeyPath string

	// BaseDomain is the base domain for the headscale server (e.g., "headscale.example.com")
	BaseDomain string

	// IPv4Prefix is the IPv4 prefix for the tailnet (e.g., "100.64.0.0/10")
	IPv4Prefix string

	// IPv6Prefix is the IPv6 prefix for the tailnet (e.g., "fd7a:115c:a1e0::/48")
	IPv6Prefix string

	// DERP configuration
	DERP DERPConfig

	// TLS configuration
	TLS TLSConfig

	// DNS configuration
	DNS DNSConfig

	// LogLevel sets the logging level (trace, debug, info, warn, error)
	LogLevel string

	// EphemeralNodeInactivityTimeout is the timeout for ephemeral nodes
	EphemeralNodeInactivityTimeout time.Duration
}

// DatabaseConfig specifies database connection parameters
type DatabaseConfig struct {
	// Type is the database type ("sqlite" or "postgres")
	Type string

	// SQLite configuration (used when Type is "sqlite")
	SQLite SQLiteConfig

	// Postgres configuration (used when Type is "postgres")
	Postgres PostgresConfig
}

// SQLiteConfig contains SQLite-specific configuration
type SQLiteConfig struct {
	// Path is the path to the SQLite database file
	Path string
}

// PostgresConfig contains PostgreSQL-specific configuration
type PostgresConfig struct {
	// Host is the PostgreSQL server host
	Host string

	// Port is the PostgreSQL server port
	Port int

	// Database is the database name
	Database string

	// Username is the database username
	Username string

	// Password is the database password
	Password string

	// SSL enables SSL connection
	SSL string

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int

	// ConnMaxIdleTimeSecs is the maximum idle time for connections in seconds
	ConnMaxIdleTimeSecs int
}

// DERPConfig contains DERP server configuration
type DERPConfig struct {
	// ServerEnabled enables the embedded DERP server
	ServerEnabled bool

	// AutomaticallyAddEmbeddedDerpRegion automatically adds the embedded DERP region
	AutomaticallyAddEmbeddedDerpRegion bool

	// ServerRegionID is the region ID for the embedded DERP server
	ServerRegionID int

	// ServerRegionCode is the region code for the embedded DERP server
	ServerRegionCode string

	// ServerRegionName is the region name for the embedded DERP server
	ServerRegionName string

	// ServerPrivateKeyPath is the path to the DERP server private key
	ServerPrivateKeyPath string

	// STUNAddr is the STUN server address
	STUNAddr string

	// URLs are external DERP server URLs
	URLs []string

	// Paths are paths to DERP map files
	Paths []string
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	// CertPath is the path to the TLS certificate file
	CertPath string

	// KeyPath is the path to the TLS private key file
	KeyPath string

	// LetsEncryptHostname enables Let's Encrypt for the specified hostname
	LetsEncryptHostname string

	// LetsEncryptCacheDir is the cache directory for Let's Encrypt certificates
	LetsEncryptCacheDir string

	// LetsEncryptChallengeType is the challenge type for Let's Encrypt
	LetsEncryptChallengeType string
}

// DNSConfig contains DNS configuration
type DNSConfig struct {
	// BaseDomain is the base domain for DNS resolution
	BaseDomain string

	// Nameservers are the DNS nameservers to use
	Nameservers []string

	// SearchDomains are the DNS search domains
	SearchDomains []string

	// ExtraRecords are additional DNS records
	ExtraRecords []DNSRecord
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	Name  string
	Type  string
	Value string
}

// ClientConfig contains configuration for connecting to a headscale control plane
type ClientConfig struct {
	// Address is the gRPC address of the headscale server
	Address string

	// APIKey is the API key for authentication (optional if using insecure connection)
	APIKey string

	// Insecure allows insecure connections (default: false)
	Insecure bool

	// Timeout is the connection timeout
	Timeout time.Duration
}
