package hclschema_test

import (
	"context"
	_ "embed"
	"fmt"
	"reflect"
	"testing"

	"github.com/fatih/color"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/walteh/hclpls/pkg/diff"
	myhclschema "github.com/walteh/hclpls/pkg/hclschema"
)

func TestJSONSchemaToReflectable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func() reflect.Type
		wantErr  bool
	}{
		{
			name: "test_basic_string_property",
			input: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name property"
					}
				}
			}`,
			expected: func() reflect.Type {
				n := struct {
					Name string `json:"name" hcl:"name,optional"`
				}{
					Name: "test",
				}
				return reflect.TypeOf(n)
			},
			wantErr: false,
		},
		{
			name:  "test_complex_server_config",
			input: string(serverConfigSchema),
			expected: func() reflect.Type {
				return toAnonymousType(TestServerConfig{})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := myhclschema.JSONSchemaToReflectable([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected error but got none")
				return
			}
			require.NoError(t, err, "unexpected error")
			AssertUnknownTypeEqual(t, tt.expected(), result)
		})
	}
}

func TestDecodeHCL(t *testing.T) {
	tests := []struct {
		name           string
		hclInput       []byte
		reflectedInput func() reflect.Type
		expected       func() interface{}
		wantErr        bool
	}{
		{
			name: "test_basic_string_property",
			expected: func() interface{} {
				v := struct {
					Name string `json:"name" hcl:"name"`
				}{
					Name: "test",
				}
				return &v
			},
			hclInput: []byte(`
				name = "test"
			`),
			reflectedInput: func() reflect.Type {
				return reflect.StructOf([]reflect.StructField{
					{
						Name: "Name",
						Tag:  reflect.StructTag(`json:"name" hcl:"name"`),
						Type: reflect.TypeOf(""),
					},
				})
			},
		},
		{
			name: "test_complex_server_config",
			expected: func() interface{} {
				return &TestServerConfig{
					Tags: []string{"prod", "api"},
					Host: "example.com",
					Port: 8443,
					Security: &TestSecurity{
						Tls: &TestTLSConfig{
							Ciphers: []string{"TLS_AES_128_GCM_SHA256"},
							Certificate: &TestCertInfo{
								Path:    "/path/to/cert.pem",
								KeyPath: "/path/to/key.pem",
							},
							Enabled: true,
							Version: "1.3",
						},
						Auth: &TestAuthConfig{
							Type: "oauth",
							Provider: &TestAuthProvider{
								Name:     "provider_name",
								Settings: map[string]string{"key": "value"},
							},
						},
						Limits: map[string]*TestRateLimit{

							"default": {
								RequestsPerSecond: 100,
								Burst:             10,
								Ips:               []string{"192.168.1.1"},
							},
						},
					},
				}
			},
			hclInput: []byte(`
				host = "example.com"
				port = 8443
				tags = ["prod", "api"]
				
				security {
					tls {
						enabled = true
						version = "1.3"
						ciphers = ["TLS_AES_128_GCM_SHA256"]
						
						certificate {
							path = "/path/to/cert.pem"
							key_path = "/path/to/key.pem"
						}
					}
					
					auth {
						type = "oauth"
						provider {
							name = "provider_name"
							settings = {
								key = "value"
							}
						}
					}
					
					limits "default" {
						requests_per_second = 100
						burst = 10
						ips = ["192.168.1.1"]
					}
				}
			`),
			reflectedInput: func() reflect.Type {
				return toAnonymousType(TestServerConfig{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			val, err := myhclschema.DecodeHCL(ctx, tt.hclInput, tt.reflectedInput())
			if tt.wantErr {
				require.Error(t, err, "expected error but got none")
				return
			}
			AssertNoDiagnostics(t, err)
			AssertUnknownValueEqualAsJSON(t, reflect.ValueOf(tt.expected()), val)
		})
	}
}

func AssertNoDiagnostics(t *testing.T, err error) {
	t.Helper()
	if diags, ok := err.(hcl.Diagnostics); ok && diags.HasErrors() {
		color.NoColor = false
		str := color.New(color.FgHiYellow, color.Faint).Sprintf("\n\n============= DIAGNOSTICS START =============\n\n")
		str += fmt.Sprintf("%s\n\n", color.YellowString("%s", t.Name()))
		for i, d := range diags {
			str += fmt.Sprintf("[%d] %s\n\n\t%s\n\n", i, d.Subject.String(), fmt.Sprintf("%s; %s", d.Summary, d.Detail))
		}
		str += color.New(color.FgHiYellow, color.Faint).Sprintf("\n\n============= DIAGNOSTICS END ===============\n\n")
		t.Log("diagnostics report:\n" + str)
		require.Fail(t, "expected no diagnostics")
	}
}

// Helper function to compare values for testing
func AssertUnknownValueEqualAsJSON(t *testing.T, expected, actual reflect.Value) {
	t.Helper()

	td := diff.TypedDiff(expected, actual)
	de := td == ""
	if !de {
		color.NoColor = false
		str := color.New(color.FgHiYellow, color.Faint).Sprintf("\n\n============= VALUE COMPARISON START =============\n\n")
		str += fmt.Sprintf("%s\n", color.YellowString("%s", t.Name()))
		str += td + "\n\n"
		str += color.New(color.FgHiYellow, color.Faint).Sprintf("============= VALUE COMPARISON END ===============\n\n")
		t.Log("value comparison report:\n" + str)
	}

	require.True(t, de, "value mismatch")
}

func AssertUnknownTypeEqual(t *testing.T, expected, actual reflect.Type) {
	t.Helper()
	td := diff.TypedDiff(expected, actual)
	de := td == ""
	if !de {
		color.NoColor = false
		str := color.New(color.FgHiYellow, color.Faint).Sprintf("\n\n============= TYPE COMPARISON START =============\n\n")
		str += fmt.Sprintf("%s\n", color.YellowString("%s", t.Name()))
		str += td + "\n\n"
		str += color.New(color.FgHiYellow, color.Faint).Sprintf("============= TYPE COMPARISON END ===============\n\n")
		t.Log("type comparison report:\n" + str)
		require.Fail(t, "type mismatch")
	}
}

func AssertKnownValueEqual[T any](t *testing.T, expected, actual T) {
	t.Helper()
	de := reflect.DeepEqual(expected, actual)
	if !de {
		color.NoColor = false
		str := color.New(color.FgHiYellow, color.Faint).Sprintf("\n\n=============  TYPED VALUE COMPARISON START =============\n\n")
		str += fmt.Sprintf("%s\n", color.YellowString("%s", t.Name()))
		str += diff.TypedDiff(expected, actual) + "\n\n"
		str += color.New(color.FgHiYellow, color.Faint).Sprintf("=============  TYPED VALUE COMPARISON END ===============\n\n")
		t.Log("value comparison report:\n" + str)
	}
	require.True(t, de, "value mismatch")
}
