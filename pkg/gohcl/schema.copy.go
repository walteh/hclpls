// 📦 generated by copyrc. DO NOT EDIT.
// 🔗 source: https://raw.githubusercontent.com/hashicorp/hcl/da7ca9a43345aab7e6785c97a1a09fdcf1d05757/gohcl/schema.go
// ℹ️ see .copyrc.lock for more details.

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gohcl

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// ImpliedBodySchema produces a hcl.BodySchema derived from the type of the
// given value, which must be a struct value or a pointer to one. If an
// inappropriate value is passed, this function will panic.
//
// The second return argument indicates whether the given struct includes
// a "remain" field, and thus the returned schema is non-exhaustive.
//
// This uses the tags on the fields of the struct to discover how each
// field's value should be expressed within configuration. If an invalid
// mapping is attempted, this function will panic.
func ImpliedBodySchema(tty reflect.Type) (schema *hcl.BodySchema, partial bool) {
	ty := tty

	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	if ty.Kind() != reflect.Struct {
		panic(fmt.Sprintf("given value must be struct, not %s", tty.String()))
	}

	var attrSchemas []hcl.AttributeSchema
	var blockSchemas []hcl.BlockHeaderSchema

	tags := getFieldTags(ty)

	attrNames := make([]string, 0, len(tags.Attributes))
	for n := range tags.Attributes {
		attrNames = append(attrNames, n)
	}
	sort.Strings(attrNames)
	for _, n := range attrNames {
		idx := tags.Attributes[n]
		optional := tags.Optional[n]
		field := ty.Field(idx)

		var required bool

		switch {
		case field.Type.AssignableTo(exprType):
			// If we're decoding to hcl.Expression then absense can be
			// indicated via a null value, so we don't specify that
			// the field is required during decoding.
			required = false
		case field.Type.Kind() != reflect.Ptr && !optional:
			required = true
		default:
			required = false
		}

		attrSchemas = append(attrSchemas, hcl.AttributeSchema{
			Name:     n,
			Required: required,
		})
	}

	blockNames := make([]string, 0, len(tags.Blocks))
	for n := range tags.Blocks {
		blockNames = append(blockNames, n)
	}
	sort.Strings(blockNames)
	for _, n := range blockNames {
		idx := tags.Blocks[n]
		field := ty.Field(idx)
		fty := field.Type
		if fty.Kind() == reflect.Slice {
			fty = fty.Elem()
		}
		if fty.Kind() == reflect.Ptr {
			fty = fty.Elem()
		}
		mapLabel := false
		if fty.Kind() != reflect.Struct {
			if fty.Kind() == reflect.Map {
				fty = fty.Elem()
				if fty.Kind() == reflect.Ptr {
					fty = fty.Elem()
				}
				mapLabel = true
			} else {
				panic(fmt.Sprintf(
					"hcl 'block' tag kind cannot be applied to %s field %s: struct required", field.Type.String(), field.Name,
				))
			}
		}
		ftags := getFieldTags(fty)
		var labelNames []string
		if len(ftags.Labels) > 0 {
			labelNames = make([]string, len(ftags.Labels))
			for i, l := range ftags.Labels {
				labelNames[i] = l.Name
			}
		}
		if mapLabel {
			labelNames = append(labelNames, "___key")
		}

		blockSchemas = append(blockSchemas, hcl.BlockHeaderSchema{
			Type:       n,
			LabelNames: labelNames,
		})
	}

	partial = tags.Remain != nil
	schema = &hcl.BodySchema{
		Attributes: attrSchemas,
		Blocks:     blockSchemas,
	}
	return schema, partial
}

type fieldTags struct {
	Attributes map[string]int
	Blocks     map[string]int
	Labels     []labelField
	Remain     *int
	Body       *int
	Optional   map[string]bool

	AttributeRange      map[string]int
	AttributeNameRange  map[string]int
	AttributeValueRange map[string]int

	DefRange   *int
	TypeRange  *int
	LabelRange map[string]int
}

type labelField struct {
	FieldIndex int
	RangeIndex int
	Name       string
}

func mapLabelKey(ty reflect.Value) reflect.Value {
	ft := getFieldTags(ty.Type())
	if ft.Labels[0].Name == "___key" {
		fmt.Print(ty.String())
		return ty.Field(ft.Labels[0].FieldIndex)
	}
	return reflect.Value{}
}

func getFieldTags(ty reflect.Type) *fieldTags {
	ret := &fieldTags{
		Attributes:          map[string]int{},
		Blocks:              map[string]int{},
		Optional:            map[string]bool{},
		AttributeRange:      map[string]int{},
		AttributeNameRange:  map[string]int{},
		AttributeValueRange: map[string]int{},
		LabelRange:          map[string]int{},
	}

	if ty.Kind() == reflect.Map {
		ty = ty.Elem()
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}
		ret.Labels = append(ret.Labels, labelField{
			FieldIndex: ty.NumField(),
			Name:       "___key",
		})
	}

	ct := ty.NumField()
	for i := 0; i < ct; i++ {
		field := ty.Field(i)
		tag := field.Tag.Get("hcl")
		if tag == "" {
			continue
		}


		split := strings.Split(tag, ",")
		name := split[0]
		kinds := split[1:]

		if !slices.Contains(kinds, "attr") && !slices.Contains(kinds, "block") && !slices.Contains(kinds, "label") {
			kinds = append(kinds, "attr")
		}

		for _, kind := range kinds {
			switch kind {
			case "attr":
				ret.Attributes[name] = i
			case "block":
			ret.Blocks[name] = i
		case "label":
			ret.Labels = append(ret.Labels, labelField{
				FieldIndex: i,
				Name:       name,
			})
		case "remain":
			if ret.Remain != nil {
				panic("only one 'remain' tag is permitted")
			}
			idx := i // copy, because this loop will continue assigning to i
			ret.Remain = &idx
		case "body":
			if ret.Body != nil {
				panic("only one 'body' tag is permitted")
			}
			idx := i // copy, because this loop will continue assigning to i
			ret.Body = &idx
		case "optional":
			// ret.Attributes[name] = i
			ret.Optional[name] = true
		case "def_range":
			if ret.DefRange != nil {
				panic("only one 'def_range' tag is permitted")
			}
			idx := i // copy, because this loop will continue assigning to i
			ret.DefRange = &idx
		case "type_range":
			if ret.TypeRange != nil {
				panic("only one 'type_range' tag is permitted")
			}
			idx := i // copy, because this loop will continue assigning to i
			ret.TypeRange = &idx
		case "label_range":
			ret.LabelRange[name] = i
		case "attr_range":
			ret.AttributeRange[name] = i
		case "attr_name_range":
			ret.AttributeNameRange[name] = i
		case "attr_value_range":
			ret.AttributeValueRange[name] = i
		default:
				panic(fmt.Sprintf("invalid hcl field tag kind %q on %s %q", kind, field.Type.String(), field.Name))
			}
		}
	}

	return ret
}
