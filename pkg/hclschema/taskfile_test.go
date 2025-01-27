package hclschema_test

import (
	_ "embed"
)

//go:embed testdata/taskfile.schema.json
var taskfileSchema []byte

//go:embed testdata/taskfile-sample-a.hcl
var taskfileHCL []byte
