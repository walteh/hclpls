package hclschema_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	myhclschema "github.com/walteh/hclpls/pkg/hclschema"
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
	Tls          *TestTLSConfig               `json:"tls" hcl:"tls,block"`
	Auth         *TestAuthConfig              `json:"auth" hcl:"auth,optional,block"`
	Limits       map[string]*TestRateLimit    `json:"limits" hcl:"limits,optional"`
	LimitWithKey []*TestRateLimitWithLabelKey `json:"limit_with_key" hcl:"limit,block"`
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

// Convert named types to anonymous structs
func toAnonymousType(v interface{}) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		return t
	}

	fields := make([]reflect.StructField, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		newField := reflect.StructField{
			Name: f.Name,
			Type: toAnonymousFieldType(f.Type),
			Tag:  f.Tag,
		}
		fields[i] = newField
	}

	return reflect.StructOf(fields)
}

// Convert named values to anonymous values
func toAnonymousValue(v interface{}) interface{} {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	t := val.Type()
	if t.Kind() != reflect.Struct {
		return v
	}

	// Create new anonymous struct with same fields
	newType := toAnonymousType(val.Interface())
	newVal := reflect.New(newType).Elem()

	// Copy fields, converting nested structs recursively
	for i := 0; i < t.NumField(); i++ {
		field := val.Field(i)
		if !field.IsValid() {
			continue
		}

		newField := newVal.Field(i)
		switch field.Kind() {
		case reflect.Ptr:
			if field.IsNil() {
				continue
			}
			converted := toAnonymousValue(field.Interface())
			if converted != nil {
				newField.Set(reflect.ValueOf(converted))
			}
		case reflect.Slice:
			if field.IsNil() {
				continue
			}
			newSlice := reflect.MakeSlice(field.Type(), field.Len(), field.Cap())
			for j := 0; j < field.Len(); j++ {
				elem := toAnonymousValue(field.Index(j).Interface())
				if elem != nil {
					newSlice.Index(j).Set(reflect.ValueOf(elem))
				}
			}
			newField.Set(newSlice)
		case reflect.Map:
			if field.IsNil() {
				continue
			}
			newMap := reflect.MakeMap(field.Type())
			iter := field.MapRange()
			for iter.Next() {
				k := iter.Key()
				v := toAnonymousValue(iter.Value().Interface())
				if v != nil {
					newMap.SetMapIndex(k, reflect.ValueOf(v))
				}
			}
			newField.Set(newMap)
		case reflect.Struct:
			converted := toAnonymousValue(field.Interface())
			if converted != nil {
				newField.Set(reflect.ValueOf(converted))
			}
		default:
			newField.Set(field)
		}
	}

	// If original was a pointer, return pointer
	if v != nil && reflect.TypeOf(v).Kind() == reflect.Ptr {
		ptr := reflect.New(newType)
		ptr.Elem().Set(newVal)
		return ptr.Interface()
	}

	return newVal.Interface()
}

func toAnonymousFieldType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr:
		return reflect.PointerTo(toAnonymousFieldType(t.Elem()))
	case reflect.Slice:
		return reflect.SliceOf(toAnonymousFieldType(t.Elem()))
	case reflect.Map:
		return reflect.MapOf(t.Key(), toAnonymousFieldType(t.Elem()))
	case reflect.Struct:
		fields := make([]reflect.StructField, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			newField := reflect.StructField{
				Name: f.Name,
				Type: toAnonymousFieldType(f.Type),
				Tag:  f.Tag,
			}
			fields[i] = newField
		}
		return reflect.StructOf(fields)
	default:
		return t
	}
}

// Helper function to compare values for testing
func compareValues(t *testing.T, expected, actual interface{}) bool {
	// Convert both to JSON and back to normalize the structures
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Logf("Failed to marshal expected value: %v", err)
		return false
	}

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Logf("Failed to marshal actual value: %v", err)
		return false
	}

	var expectedMap, actualMap map[string]interface{}
	if err := json.Unmarshal(expectedJSON, &expectedMap); err != nil {
		t.Logf("Failed to unmarshal expected value: %v", err)
		return false
	}

	if err := json.Unmarshal(actualJSON, &actualMap); err != nil {
		t.Logf("Failed to unmarshal actual value: %v", err)
		return false
	}

	de := reflect.DeepEqual(expectedMap, actualMap)
	if !de {
		dmp := diffmatchpatch.New()
		expStr := fmt.Sprintf("%#v", expectedMap)
		actStr := fmt.Sprintf("%#v", actualMap)
		diffs := dmp.DiffMain(expStr, actStr, false)
		t.Log("============= VALUE COMPARISON START =============")
		t.Log("expected:", expStr)
		t.Log("actual:", actStr)
		t.Log("diff:", dmp.DiffPrettyText(diffs))
		t.Log("============= VALUE COMPARISON END =============")
	}
	return de
}

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
					Name string `json:"name" hcl:"name"`
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
			de := reflect.DeepEqual(tt.expected(), result)
			assert.True(t, de, "schema conversion mismatch")
			if !de {
				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(tt.expected().String(), result.String(), false)
				t.Log("============= TYPE COMPARISON START =============")
				t.Log("expected:", tt.expected().String())
				t.Log("actual:", result.String())
				t.Log("diff:", dmp.DiffPrettyText(diffs))
				t.Log("============= TYPE COMPARISON END =============")
			}
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
						LimitWithKey: []*TestRateLimitWithLabelKey{
							{
								Key:               "default",
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
					
					limit "default" {
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
			assert.NoError(t, err, "unexpected error")
			if diags, ok := err.(hcl.Diagnostics); ok && len(diags) > 0 {
				t.Log("============= DIAGNOSTICS START =============")
				for _, diag := range diags {
					t.Log("diag", diag.Error())
				}
				t.Log("============= DIAGNOSTICS END ===============")
			}
			assert.True(t, compareValues(t, tt.expected(), val.Interface()), "value mismatch")
		})
	}
}

//go:embed testdata/taskfile.schema.json
var taskfileSchema []byte

//go:embed testdata/taskfile-sample-a.hcl
var taskfileHCL []byte
