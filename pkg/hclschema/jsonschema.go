package hclschema

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/walteh/hclpls/pkg/gohcl"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/hclpls/pkg/generate"
)

func DecodeHCL(ctx context.Context, hcls []byte, typ reflect.Type) (reflect.Value, error) {

	file, diags := hclsyntax.ParseConfig(hcls, "schema.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return reflect.Value{}, errors.Errorf("parsing hcl: %w", diags)
	}

	ectx := &hcl.EvalContext{}

	val := reflect.New(typ)

	fmt.Println("val", typ.String())

	diags = gohcl.DecodeBodyToStruct(file.Body, ectx, val.Elem(), typ)
	if diags.HasErrors() {
		return reflect.Value{}, diags
	}

	return val, nil
}

// JSONSchemaToHCL converts a JSON schema into an HCL schema
// 🔄 This function uses go-jsonschema to generate Go structs,
// then converts them to HCL schema via reflection
func JSONSchemaToReflectable(jsonSchema []byte) (reflect.Type, error) {

	uri := url.URL{
		Scheme: "file",
		Path:   "schema.json",
	}

	// Generate Go code from JSON schema
	schema, err := generate.ParseWithSchemaKeyRequired(string(jsonSchema), &uri, false)
	if err != nil {
		return nil, errors.Errorf("generating schema: %w", err)
	}

	gen := generate.New(schema)

	err = gen.CreateTypes()
	if err != nil {
		return nil, errors.Errorf("creating types: %w", err)
	}

	reflectable, err := ToReflectableStruct(gen)
	if err != nil {
		return nil, errors.Errorf("converting schema to reflectable: %w", err)
	}

	return reflectable, nil

}
