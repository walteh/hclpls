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
		return reflect.Value{}, errors.Errorf("decoding hcl: %w", diags)
	}

	return val, nil
}

// JSONSchemaToHCL converts a JSON schema into an HCL schema
// 🔄 This function uses go-jsonschema to generate Go structs,
// then converts them to HCL schema via reflection
func JSONSchemaToReflectable(jsonSchema []byte) (reflect.Value, error) {

	uri := url.URL{
		Scheme: "file",
		Path:   "schema.json",
	}

	// Generate Go code from JSON schema
	schema, err := generate.Parse(string(jsonSchema), &uri)
	if err != nil {
		return reflect.Value{}, errors.Errorf("generating schema: %w", err)
	}

	gen := generate.New(schema)

	reflectable, err := gen.ToReflectableStruct()
	if err != nil {
		return reflect.Value{}, errors.Errorf("converting schema to reflectable: %w", err)
	}

	return reflectable, nil

	// outputWriter := bytes.NewBuffer(nil)

	// generate.Output(outputWriter, gen, "schema")

	// data := outputWriter.Bytes()

	// // Parse the generated code into AST
	// fset := token.NewFileSet()
	// file, err := parser.ParseFile(fset, "schema.go", data, parser.ParseComments)
	// if err != nil {
	// 	return reflect.Value{}, errors.Errorf("parsing generated code: %w", err)
	// }

	// // Convert AST to reflect types
	// types, err := astToReflectTypes(file)
	// if err != nil {
	// 	return reflect.Value{}, errors.Errorf("converting ast to types: %w", err)
	// }

	// schemas, ok := gohcl.ImpliedBodySchema(types)
	// if !ok {
	// 	return reflect.Value{}, errors.New("generating implied body schema")
	// }

	// return reflect.ValueOf(schemas), nil
}
