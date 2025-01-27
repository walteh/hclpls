package hclschema_test

import (
	_ "embed"
)

//go:embed testdata/docker-bake.schema.json
var dockerbakeSchema []byte

//go:embed testdata/docker-bake.sample-a.hcl
var dockerbakeHCL []byte
