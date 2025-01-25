

copy {
	source {
		repo = "github.com/a-h/generate"
		ref  = "master"
		path = "."
	}
	destination {
		path = "./pkg/generate"
	}
	options {
		replacements = [

		]
		ignore_files = [
			".gitignore",
			".gitattributes",
			".github",
			"*.txt",
			"Makefile",
			"*.json",
			".travis.yml",
		]
	}
}

copy {
	source {
		repo = "github.com/hashicorp/hcl"
		ref  = "main"
		path = "gohcl"
	}
	destination {
		path = "./pkg/gohcl"
	}
	options {
		replacements = [
			{
				old = "github.com/hashicorp/hcl/v2/gohcl",
				new = "github.com/walteh/hclpls/pkg/gohcl",
			},
			{
				old = "func ImpliedBodySchema(val interface{}) (schema *hcl.BodySchema, partial bool) {\n\tty := reflect.TypeOf(val)",
				new = "func ImpliedBodySchema(tty reflect.Type) (schema *hcl.BodySchema, partial bool) {\n\tty := tty",
			},
			{
				old = "be struct, not %T\", val))",
				new = "be struct, not %s\", tty.String()))",
			},
			{
				old = <<EOT
func decodeBodyToStruct(body hcl.Body, ctx *hcl.EvalContext, val reflect.Value) hcl.Diagnostics {
	schema, partial := ImpliedBodySchema(val.Interface())
EOT
				new = <<EOT
func DecodeBodyToStruct(body hcl.Body, ctx *hcl.EvalContext, val reflect.Value, tty reflect.Type) hcl.Diagnostics {
	schema, partial := ImpliedBodySchema(tty)
EOT
			},
			{
				old = "return decodeBodyToStruct(body, ctx, val)",
				new = "return DecodeBodyToStruct(body, ctx, val, val.Type())",
			},
		]
		ignore_files = [
			"**/*_test.go",
		]
	}
}
