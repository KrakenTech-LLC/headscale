package controlplane

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/rs/zerolog"
	"tailscale.com/tailcfg"
	"tailscale.com/types/dnstype"
)

// DefaultServerConfig returns a ServerConfig with sensible defaults
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerURL:         "http://localhost:8080",
		ListenAddr:        "0.0.0.0:8080",
		GRPCAddr:          "0.0.0.0:50443",
		GRPCAllowInsecure: true,
		Database: DatabaseConfig{
			Type: "sqlite",
			SQLite: SQLiteConfig{
				Path: "/tmp/headscale.db",
			},
		},
		NoisePrivateKeyPath: "/tmp/headscale_noise_private.key",
		BaseDomain:          "headscale.local",
		IPv4Prefix:          "100.64.0.0/10",
		IPv6Prefix:          "fd7a:115c:a1e0::/48",
		DERP: DERPConfig{
			ServerEnabled:                      true,
			AutomaticallyAddEmbeddedDerpRegion: true,
			ServerRegionID:                     999,
			ServerRegionCode:                   "headscale",
			ServerRegionName:                   "Headscale Embedded DERP",
			ServerPrivateKeyPath:               "/tmp/headscale_derp_private.key",
			STUNAddr:                           "0.0.0.0:3478",
		},
		DNS: DNSConfig{
			BaseDomain:    "headscale.local",
			Nameservers:   []string{"1.1.1.1", "8.8.8.8"},
			SearchDomains: []string{},
		},
		LogLevel:                       "info",
		EphemeralNodeInactivityTimeout: time.Hour * 24 * 30, // 30 days
	}
}

// ToHeadscaleConfig converts a ServerConfig to the internal headscale types.Config
func (sc *ServerConfig) ToHeadscaleConfig() (*types.Config, error) {
	// Parse IPv4 prefix
	var prefixV4 *netip.Prefix
	if sc.IPv4Prefix != "" {
		prefix, err := netip.ParsePrefix(sc.IPv4Prefix)
		if err != nil {
			return nil, fmt.Errorf("invalid IPv4 prefix %q: %w", sc.IPv4Prefix, err)
		}
		prefixV4 = &prefix
	}

	// Parse IPv6 prefix
	var prefixV6 *netip.Prefix
	if sc.IPv6Prefix != "" {
		prefix, err := netip.ParsePrefix(sc.IPv6Prefix)
		if err != nil {
			return nil, fmt.Errorf("invalid IPv6 prefix %q: %w", sc.IPv6Prefix, err)
		}
		prefixV6 = &prefix
	}

	// Convert log level
	logLevel, err := parseLogLevel(sc.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", sc.LogLevel, err)
	}

	// Build database config
	dbConfig, err := sc.buildDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("building database config: %w", err)
	}

	// Build DERP config
	derpConfig := sc.buildDERPConfig()

	// Build TLS config
	tlsConfig := sc.buildTLSConfig()

	// Build DNS config
	dnsConfig := sc.buildDNSConfig()

	config := &types.Config{
		ServerURL:                      sc.ServerURL,
		Addr:                           sc.ListenAddr,
		GRPCAddr:                       sc.GRPCAddr,
		GRPCAllowInsecure:              sc.GRPCAllowInsecure,
		NoisePrivateKeyPath:            sc.NoisePrivateKeyPath,
		BaseDomain:                     sc.BaseDomain,
		PrefixV4:                       prefixV4,
		PrefixV6:                       prefixV6,
		IPAllocation:                   types.IPAllocationStrategySequential,
		EphemeralNodeInactivityTimeout: sc.EphemeralNodeInactivityTimeout,
		Database:                       dbConfig,
		DERP:                           derpConfig,
		TLS:                            tlsConfig,
		DNSConfig:                      dnsConfig,
		TailcfgDNSConfig:               dnsToTailcfgDNS(dnsConfig),
		UnixSocket:                     "/tmp/headscale.sock",
		UnixSocketPermission:           0o770,
		DisableUpdateCheck:             true,
		RandomizeClientPort:            false,
		Log: types.LogConfig{
			Format: "text",
			Level:  logLevel,
		},
		Policy: types.PolicyConfig{
			Mode: types.PolicyModeFile,
			Path: "",
		},
		CLI: types.CLIConfig{
			Address:  sc.GRPCAddr,
			Insecure: sc.GRPCAllowInsecure,
			Timeout:  30 * time.Second,
		},
	}

	return config, nil
}

// buildDatabaseConfig converts the database configuration
func (sc *ServerConfig) buildDatabaseConfig() (types.DatabaseConfig, error) {
	switch sc.Database.Type {
	case "sqlite":
		return types.DatabaseConfig{
			Type: "sqlite",
			Sqlite: types.SqliteConfig{
				Path: sc.Database.SQLite.Path,
			},
		}, nil
	case "postgres":
		return types.DatabaseConfig{
			Type: "postgres",
			Postgres: types.PostgresConfig{
				Host:                sc.Database.Postgres.Host,
				Port:                sc.Database.Postgres.Port,
				Name:                sc.Database.Postgres.Database,
				User:                sc.Database.Postgres.Username,
				Pass:                sc.Database.Postgres.Password,
				Ssl:                 sc.Database.Postgres.SSL,
				MaxOpenConnections:  sc.Database.Postgres.MaxOpenConns,
				MaxIdleConnections:  sc.Database.Postgres.MaxIdleConns,
				ConnMaxIdleTimeSecs: sc.Database.Postgres.ConnMaxIdleTimeSecs,
			},
		}, nil
	default:
		return types.DatabaseConfig{}, fmt.Errorf("unsupported database type: %s", sc.Database.Type)
	}
}

// buildDERPConfig converts the DERP configuration
func (sc *ServerConfig) buildDERPConfig() types.DERPConfig {
	var serverUrls []url.URL
	for _, urlStr := range sc.DERP.URLs {
		url, err := url.Parse(urlStr)
		if err != nil {
			continue
		}
		serverUrls = append(serverUrls, *url)
	}

	return types.DERPConfig{
		ServerEnabled:                      sc.DERP.ServerEnabled,
		AutomaticallyAddEmbeddedDerpRegion: sc.DERP.AutomaticallyAddEmbeddedDerpRegion,
		ServerRegionID:                     sc.DERP.ServerRegionID,
		ServerRegionCode:                   sc.DERP.ServerRegionCode,
		ServerRegionName:                   sc.DERP.ServerRegionName,
		ServerPrivateKeyPath:               sc.DERP.ServerPrivateKeyPath,
		STUNAddr:                           sc.DERP.STUNAddr,
		URLs:                               serverUrls,
		Paths:                              sc.DERP.Paths,
		AutoUpdate:                         false,
		UpdateFrequency:                    24 * time.Hour,
	}
}

// buildTLSConfig converts the TLS configuration
func (sc *ServerConfig) buildTLSConfig() types.TLSConfig {
	return types.TLSConfig{
		CertPath: sc.TLS.CertPath,
		KeyPath:  sc.TLS.KeyPath,
		LetsEncrypt: types.LetsEncryptConfig{
			Hostname:      sc.TLS.LetsEncryptHostname,
			Listen:        "", // Not exposed in our simplified config
			CacheDir:      sc.TLS.LetsEncryptCacheDir,
			ChallengeType: sc.TLS.LetsEncryptChallengeType,
		},
	}
}

// buildDNSConfig converts the DNS configuration
func (sc *ServerConfig) buildDNSConfig() types.DNSConfig {
	return types.DNSConfig{
		MagicDNS:         true,
		BaseDomain:       sc.DNS.BaseDomain,
		OverrideLocalDNS: true,
		Nameservers: types.Nameservers{
			Global: sc.DNS.Nameservers,
			Split:  map[string][]string{},
		},
		SearchDomains: sc.DNS.SearchDomains,
		ExtraRecords:  []tailcfg.DNSRecord{}, // Convert if needed
	}
}

// parseLogLevel converts string log level to zerolog level
func parseLogLevel(level string) (zerolog.Level, error) {
	switch level {
	case "trace":
		return zerolog.TraceLevel, nil
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// dnsToTailcfgDNS converts DNS config to tailcfg format using the actual headscale implementation
func dnsToTailcfgDNS(dns types.DNSConfig) *tailcfg.DNSConfig {
	cfg := tailcfg.DNSConfig{}

	if dns.BaseDomain == "" && dns.MagicDNS {
		// Don't fatal here, just log a warning since this is a library
		fmt.Printf("Warning: dns.base_domain must be set when using MagicDNS\n")
	}

	cfg.Proxied = dns.MagicDNS
	cfg.ExtraRecords = dns.ExtraRecords

	// Use the actual headscale implementation for resolvers
	globalResolvers := globalResolvers(dns)
	if dns.OverrideLocalDNS {
		cfg.Resolvers = globalResolvers
	} else {
		cfg.FallbackResolvers = globalResolvers
	}

	routes := splitResolvers(dns)
	cfg.Routes = routes
	if dns.BaseDomain != "" {
		cfg.Domains = []string{dns.BaseDomain}
	}
	cfg.Domains = append(cfg.Domains, dns.SearchDomains...)

	return &cfg
}

// globalResolvers returns the global DNS resolvers from the headscale implementation
func globalResolvers(d types.DNSConfig) []*dnstype.Resolver {
	var resolvers []*dnstype.Resolver

	for _, nsStr := range d.Nameservers.Global {
		warn := ""
		if _, err := netip.ParseAddr(nsStr); err == nil {
			resolvers = append(resolvers, &dnstype.Resolver{
				Addr: nsStr,
			})
			continue
		} else {
			warn = fmt.Sprintf("Invalid global nameserver %q. Parsing error: %s ignoring", nsStr, err)
		}

		if _, err := url.Parse(nsStr); err == nil {
			resolvers = append(resolvers, &dnstype.Resolver{
				Addr: nsStr,
			})
			continue
		} else {
			warn = fmt.Sprintf("Invalid global nameserver %q. Parsing error: %s ignoring", nsStr, err)
		}

		if warn != "" {
			fmt.Printf("Warning: %s\n", warn)
		}
	}

	return resolvers
}

// splitResolvers returns a map of domain to DNS resolvers from the headscale implementation
func splitResolvers(d types.DNSConfig) map[string][]*dnstype.Resolver {
	routes := make(map[string][]*dnstype.Resolver)
	for domain, nameservers := range d.Nameservers.Split {
		var resolvers []*dnstype.Resolver
		for _, nsStr := range nameservers {
			warn := ""
			if _, err := netip.ParseAddr(nsStr); err == nil {
				resolvers = append(resolvers, &dnstype.Resolver{
					Addr: nsStr,
				})
				continue
			} else {
				warn = fmt.Sprintf("Invalid split dns nameserver %q. Parsing error: %s ignoring", nsStr, err)
			}

			if _, err := url.Parse(nsStr); err == nil {
				resolvers = append(resolvers, &dnstype.Resolver{
					Addr: nsStr,
				})
				continue
			} else {
				warn = fmt.Sprintf("Invalid split dns nameserver %q. Parsing error: %s ignoring", nsStr, err)
			}

			if warn != "" {
				fmt.Printf("Warning: %s\n", warn)
			}
		}
		routes[domain] = resolvers
	}

	return routes
}

// EnsureDirectories creates necessary directories for the configuration
func (sc *ServerConfig) EnsureDirectories() error {
	dirs := []string{
		filepath.Dir(sc.NoisePrivateKeyPath),
		filepath.Dir(sc.DERP.ServerPrivateKeyPath),
	}

	if sc.Database.Type == "sqlite" {
		dirs = append(dirs, filepath.Dir(sc.Database.SQLite.Path))
	}

	if sc.TLS.LetsEncryptCacheDir != "" {
		dirs = append(dirs, sc.TLS.LetsEncryptCacheDir)
	}

	for _, dir := range dirs {
		if dir != "" && dir != "." && dir != "/" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	return nil
}

// Validate checks if the configuration is valid
func (sc *ServerConfig) Validate() error {
	if sc.ServerURL == "" {
		return fmt.Errorf("ServerURL is required")
	}

	if sc.NoisePrivateKeyPath == "" {
		return fmt.Errorf("NoisePrivateKeyPath is required")
	}

	if sc.BaseDomain == "" {
		return fmt.Errorf("BaseDomain is required")
	}

	if sc.IPv4Prefix == "" && sc.IPv6Prefix == "" {
		return fmt.Errorf("at least one of IPv4Prefix or IPv6Prefix is required")
	}

	if sc.Database.Type == "" {
		return fmt.Errorf("Database.Type is required")
	}

	if sc.Database.Type == "sqlite" && sc.Database.SQLite.Path == "" {
		return fmt.Errorf("Database.SQLite.Path is required when using SQLite")
	}

	if sc.Database.Type == "postgres" {
		if sc.Database.Postgres.Host == "" {
			return fmt.Errorf("Database.Postgres.Host is required when using PostgreSQL")
		}
		if sc.Database.Postgres.Database == "" {
			return fmt.Errorf("Database.Postgres.Database is required when using PostgreSQL")
		}
		if sc.Database.Postgres.Username == "" {
			return fmt.Errorf("Database.Postgres.Username is required when using PostgreSQL")
		}
	}

	return nil
}

// DefaultClientConfig returns a ClientConfig with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Address:  "localhost:50443",
		Insecure: true,
		Timeout:  30 * time.Second,
	}
}
