package hclschema_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walteh/hclpls/pkg/gohcl"
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
					Name string `json:"name" hcl:"name"`
				}{
					Name: "test",
				}
				v := struct {
					Root struct {
						Name string `json:"name" hcl:"name"`
					} `json:"Root" hcl:"Root"`
				}{
					Root: n,
				}
				return reflect.TypeOf(v)
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
			require.Equal(t, tt.expected().String(), result.String(), "schema conversion mismatch")
		})
	}
}

func TestDecodeHCL(t *testing.T) {
	tests := []struct {
		name           string
		hclInput       []byte
		reflectedInput func() reflect.Type
		expected       func() reflect.Value
		wantErr        bool
	}{
		{
			name: "test_basic_string_property",
			expected: func() reflect.Value {
				v := struct {
					Name string `json:"name" hcl:"name"`
				}{
					Name: "test",
				}
				return reflect.ValueOf(&v)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			schema, partial := gohcl.ImpliedBodySchema(tt.reflectedInput())
			fmt.Println("schema", schema)
			fmt.Println("partial", partial)
			val, err := myhclschema.DecodeHCL(ctx, tt.hclInput, tt.reflectedInput())
			require.NoError(t, err, "unexpected error")
			require.Equal(t, tt.expected().Interface(), val.Interface(), "decoded value mismatch")
		})
	}
}
