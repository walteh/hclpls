package hclschema_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	myhclschema "github.com/walteh/hclpls/pkg/hclschema"
)

//go:embed testdata/docker-bake.schema.json
var dockerbakeSchema []byte

//go:embed testdata/docker-bake.sample-a.hcl
var dockerbakeHCL []byte

//go:embed testdata/docker-bake.sample-a.json
var dockerbakeJSON []byte

func TestDockerBake(t *testing.T) {
	ctx := context.Background()
	result, err := myhclschema.JSONSchemaToReflectable(dockerbakeSchema)
	require.NoError(t, err)

	fmt.Println(result.String())

	val, err := myhclschema.DecodeHCL(ctx, dockerbakeHCL, result)
	AssertNoDiagnostics(t, err)

	var expected any
	require.NoError(t, json.Unmarshal(dockerbakeJSON, &expected))

	AssertUnknownValueEqualAsJSON(t, reflect.ValueOf(expected), val)
}
