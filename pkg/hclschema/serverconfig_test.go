package hclschema_test

import (
	_ "embed"
)

//go:embed testdata/server-config.schema.json
var serverConfigSchema []byte

// Named types for readability
type TestServerConfig struct {
	Tags     []string      `json:"tags" hcl:"tags,optional"`
	Security *TestSecurity `json:"security" hcl:"security,block"`
	Host     string        `json:"host" hcl:"host"`
	Port     int           `json:"port" hcl:"port"`
}

type TestSecurity struct {
	Tls    *TestTLSConfig            `json:"tls" hcl:"tls,block"`
	Auth   *TestAuthConfig           `json:"auth" hcl:"auth,optional,block"`
	Limits map[string]*TestRateLimit `json:"limits" hcl:"limits,optional,block"`
}

type TestTLSConfig struct {
	Ciphers     []string      `json:"ciphers" hcl:"ciphers,optional"`
	Certificate *TestCertInfo `json:"certificate" hcl:"certificate,block"`
	Enabled     bool          `json:"enabled" hcl:"enabled"`
	Version     string        `json:"version" hcl:"version"`
}

type TestCertInfo struct {
	Path    string `json:"path" hcl:"path"`
	KeyPath string `json:"key_path" hcl:"key_path,optional"`
}

type TestAuthConfig struct {
	Type     string            `json:"type" hcl:"type"`
	Provider *TestAuthProvider `json:"provider" hcl:"provider,block"`
}

type TestAuthProvider struct {
	Settings map[string]string `json:"settings" hcl:"settings,optional"`
	Name     string            `json:"name" hcl:"name"`
}

type TestRateLimit struct {
	RequestsPerSecond int      `json:"requests_per_second" hcl:"requests_per_second"`
	Burst             int      `json:"burst" hcl:"burst,optional"`
	Ips               []string `json:"ips" hcl:"ips,optional"`
}

type TestRateLimitWithLabelKey struct {
	RequestsPerSecond int      `json:"requests_per_second" hcl:"requests_per_second"`
	Burst             int      `json:"burst" hcl:"burst,optional"`
	Ips               []string `json:"ips" hcl:"ips,optional"`
	Key               string   `json:"key" hcl:"key,label"`
}
