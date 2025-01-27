// 📦 generated by copyrc. DO NOT EDIT.
// 🔗 source: https://raw.githubusercontent.com/hashicorp/hcl/da7ca9a43345aab7e6785c97a1a09fdcf1d05757/gohcl/decode.go
// ℹ️ see .copyrc.lock for more details.

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gohcl

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"
)

// DecodeBody extracts the configuration within the given body into the given
// value. This value must be a non-nil pointer to either a struct or
// a map, where in the former case the configuration will be decoded using
// struct tags and in the latter case only attributes are allowed and their
// values are decoded into the map.
//
// The given EvalContext is used to resolve any variables or functions in
// expressions encountered while decoding. This may be nil to require only
// constant values, for simple applications that do not support variables or
// functions.
//
// The returned diagnostics should be inspected with its HasErrors method to
// determine if the populated value is valid and complete. If error diagnostics
// are returned then the given value may have been partially-populated but
// may still be accessed by a careful caller for static analysis and editor
// integration use-cases.
func DecodeBody(body hcl.Body, ctx *hcl.EvalContext, val interface{}) hcl.Diagnostics {
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("target value must be a pointer, not %s", rv.Type().String()))
	}

	return decodeBodyToValue(body, ctx, rv.Elem())
}

func decodeBodyToValue(body hcl.Body, ctx *hcl.EvalContext, val reflect.Value) hcl.Diagnostics {
	et := val.Type()
	switch et.Kind() {
	case reflect.Struct:
		return DecodeBodyToStruct(body, ctx, val, val.Type())
	case reflect.Map:
		return decodeBodyToMap(body, ctx, val)
	default:
		panic(fmt.Sprintf("target value must be pointer to struct or map, not %s", et.String()))
	}
}

func DecodeBodyToStruct(body hcl.Body, ctx *hcl.EvalContext, val reflect.Value, tty reflect.Type) hcl.Diagnostics {
	schema, partial := ImpliedBodySchema(tty)

	var content *hcl.BodyContent
	var leftovers hcl.Body
	var diags hcl.Diagnostics
	if partial {
		content, leftovers, diags = body.PartialContent(schema)
	} else {
		content, diags = body.Content(schema)
	}
	if content == nil {
		return diags
	}

	tags := getFieldTags(val.Type())

	if tags.Body != nil {
		fieldIdx := *tags.Body
		field := val.Type().Field(fieldIdx)
		fieldV := val.Field(fieldIdx)
		switch {
		case bodyType.AssignableTo(field.Type):
			fieldV.Set(reflect.ValueOf(body))

		default:
			diags = append(diags, decodeBodyToValue(body, ctx, fieldV)...)
		}
	}

	if tags.Remain != nil {
		fieldIdx := *tags.Remain
		field := val.Type().Field(fieldIdx)
		fieldV := val.Field(fieldIdx)
		switch {
		case bodyType.AssignableTo(field.Type):
			fieldV.Set(reflect.ValueOf(leftovers))
		case attrsType.AssignableTo(field.Type):
			attrs, attrsDiags := leftovers.JustAttributes()
			if len(attrsDiags) > 0 {
				diags = append(diags, attrsDiags...)
			}
			fieldV.Set(reflect.ValueOf(attrs))
		default:
			diags = append(diags, decodeBodyToValue(leftovers, ctx, fieldV)...)
		}
	}

	for name, fieldIdx := range tags.Attributes {
		attr := content.Attributes[name]
		field := val.Type().Field(fieldIdx)
		fieldV := val.Field(fieldIdx)

		if attr == nil {
			if !exprType.AssignableTo(field.Type) {
				continue
			}

			// As a special case, if the target is of type hcl.Expression then
			// we'll assign an actual expression that evaluates to a cty null,
			// so the caller can deal with it within the cty realm rather
			// than within the Go realm.
			synthExpr := hcl.StaticExpr(cty.NullVal(cty.DynamicPseudoType), body.MissingItemRange())
			fieldV.Set(reflect.ValueOf(synthExpr))
			continue
		}

		if attrRange, exists := tags.AttributeRange[name]; exists {
			val.Field(attrRange).Set(reflect.ValueOf(attr.Range))
		}

		if attrNameRange, exists := tags.AttributeNameRange[name]; exists {
			val.Field(attrNameRange).Set(reflect.ValueOf(attr.NameRange))
		}

		if attrValueRange, exists := tags.AttributeValueRange[name]; exists {
			val.Field(attrValueRange).Set(reflect.ValueOf(attr.Expr.Range()))
		}
		fmt.Println(name, field.Type.String(), fieldV.Type().String(), attrType.String())
		switch {
		case attrType.AssignableTo(field.Type):
			fieldV.Set(reflect.ValueOf(attr))
		case exprType.AssignableTo(field.Type):
			fieldV.Set(reflect.ValueOf(attr.Expr))
		default:
			diags = append(diags, DecodeExpression(
				attr.Expr, ctx, fieldV.Addr().Interface(),
			)...)
		}
	}

	blocksByType := content.Blocks.ByType()

	for typeName, fieldIdx := range tags.Blocks {
		blocks := blocksByType[typeName]
		field := val.Type().Field(fieldIdx)

		ty := field.Type
		isSlice := false
		isPtr := false
		isMap := false
		if ty.Kind() == reflect.Slice {
			isSlice = true
			ty = ty.Elem()
		}
		if ty.Kind() == reflect.Ptr {
			isPtr = true
			ty = ty.Elem()
		}
		if ty.Kind() == reflect.Map {
			isMap = true
			ty = ty.Elem()
			if ty.Kind() == reflect.Ptr {
				isPtr = true
				ty = ty.Elem()
			}
		}

		if len(blocks) > 1 && !isSlice && !isMap {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate %s block", typeName),
				Detail: fmt.Sprintf(
					"Only one %s block is allowed. Another was defined at %s.",
					typeName, blocks[0].DefRange.String(),
				),
				Subject: &blocks[1].DefRange,
			})
			continue
		}

		if len(blocks) == 0 {
			if isSlice || isPtr || isMap {
				if val.Field(fieldIdx).IsNil() {
					val.Field(fieldIdx).Set(reflect.Zero(field.Type))
				}
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Missing %s block", typeName),
					Detail:   fmt.Sprintf("A %s block is required.", typeName),
					Subject:  body.MissingItemRange().Ptr(),
				})
			}
			continue
		}

		switch {

		case isSlice:
			elemType := ty
			if isPtr {
				elemType = reflect.PtrTo(ty)
			}
			sli := val.Field(fieldIdx)
			if sli.IsNil() {
				sli = reflect.MakeSlice(reflect.SliceOf(elemType), len(blocks), len(blocks))
			}

			for i, block := range blocks {
				if isPtr {
					if i >= sli.Len() {
						sli = reflect.Append(sli, reflect.New(ty))
					}
					v := sli.Index(i)
					if v.IsNil() {
						v = reflect.New(ty)
					}
					diags = append(diags, decodeBlockToValue(block, ctx, v.Elem())...)
					sli.Index(i).Set(v)
				} else {
					if i >= sli.Len() {
						sli = reflect.Append(sli, reflect.Indirect(reflect.New(ty)))
					}
					diags = append(diags, decodeBlockToValue(block, ctx, sli.Index(i))...)
				}
			}

			if sli.Len() > len(blocks) {
				sli.SetLen(len(blocks))
			}

			val.Field(fieldIdx).Set(sli)
		case isMap:
			elemType := ty
			if isPtr {
				elemType = reflect.PointerTo(ty)
			}
			sli := val.Field(fieldIdx)
			if sli.IsNil() {
				sli = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), elemType))
			}
// "reflect.Value.SetMapIndex: value of type struct { RequestsPerSecond int \"json:\\\"requests_per_second\\\" hcl:\\\"requests_per_second\\\"\"; Burst int \"json:\\\"burst\\\" hcl:\\\"burst,optional\\\"\"; Ips []string \"json:\\\"ips\\\" hcl:\\\"ips,optional\\\"\" } is not assignable to type *struct { RequestsPerSecond int \"json:\\\"requests_per_second\\\" hcl:\\\"requests_per_second\\\"\"; Burst int \"json:\\\"burst\\\" hcl:\\\"burst,optional\\\"\"; Ips []string \"json:\\\"ips\\\" hcl:\\\"ips,optional\\\"\" }"
			for _, block := range blocks {
				keyname := block.Labels[0]
				if isPtr {
					// if i >= sli.Len() {
					// 	sli = reflect.Append(sli, reflect.New(ty))
					// }
					// v := sli.MapIndex(reflect.ValueOf(keyname))
					// // v := sli.Index(i)
					// if v.IsNil() {
					// }
					v := reflect.New(ty)
					fmt.Println("decodeBodyToValue", v.Type().String())

					diags = append(diags, decodeBodyToValue(block.Body, ctx, v.Elem())...)
					fmt.Println("decodeBodyToValue a", v.Elem().Type().String())
					fmt.Println("decodeBodyToValue b", sli.Type().String())
					fmt.Println("decodeBodyToValue c", v.Type().String())
					sli.SetMapIndex(reflect.ValueOf(keyname), v)
				} else {
					// if i >= sli.Len() {
					// 	sli.MapIndex(reflect.ValueOf(keyname)).Set(v)

					// 	sli = reflect.Append(sli, reflect.Indirect(reflect.New(ty)))
					// }
					v := sli.MapIndex(reflect.ValueOf(keyname))
					if !v.IsValid() {
						v = reflect.New(ty)
					}
					diags = append(diags, decodeBodyToValue(block.Body, ctx, v)...)
					sli.MapIndex(reflect.ValueOf(keyname)).Set(v)
				}
			}

			if sli.Len() > len(blocks) {
				sli.SetLen(len(blocks))
			}

			val.Field(fieldIdx).Set(sli)
		default:
			block := blocks[0]
			if isPtr {
				v := val.Field(fieldIdx)
				if v.IsNil() {
					v = reflect.New(ty)
				}
				diags = append(diags, decodeBlockToValue(block, ctx, v.Elem())...)
				val.Field(fieldIdx).Set(v)
			} else {
							diags = append(diags, decodeBlockToValue(block, ctx, val.Field(fieldIdx))...)
			}

		}

	}

	return diags
}

func decodeBodyToMap(body hcl.Body, ctx *hcl.EvalContext, v reflect.Value) hcl.Diagnostics {

	fmt.Println("decodeBodyToMap", v.Type().String())
	attrs, diags := body.JustAttributes()
	if attrs == nil {
		return diags
	}

	mv := reflect.MakeMap(v.Type())

	for k, attr := range attrs {
		switch {
		case attrType.AssignableTo(v.Type().Elem()):
			mv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(attr))
		case exprType.AssignableTo(v.Type().Elem()):
			mv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(attr.Expr))
		default:
			ev := reflect.New(v.Type().Elem())
			// this just triggers an error, no logical reason for it
			// diags = append(diags, DecodeExpression(attr.Expr, ctx, ev.Interface())...)
			mv.SetMapIndex(reflect.ValueOf(k), ev.Elem())
		}
	}

	v.Set(mv)

	return diags
}

// func decodeBlockToMapValue(block *hcl.Block, ctx *hcl.EvalContext, v reflect.Value) hcl.Diagnostics {
// 	diags := decodeBodyToValue(block.Body, ctx, v)
// 	// wrap tupe 
// 	blockTags := getFieldTags(v.Type())

// }

func decodeBlockToValue(block *hcl.Block, ctx *hcl.EvalContext, v reflect.Value) hcl.Diagnostics {
	diags := decodeBodyToValue(block.Body, ctx, v)

	blockTags := getFieldTags(v.Type())
	// if we have a ___key, we need to set the map index to the value of the map
	// for li1, _ := range block.Labels {
	// 	lfieldName := blockTags.Labels[li1].Name
	// 	if lfieldName == "___key" {
	// 		return diags
	// 		// fmt.Println("keyv", keyv)
	// 		// pp.Println("v", v.Interface())

	// 		// realv := v.MapIndex(reflect.ValueOf(keyv))
	// 		// // v.MapIndex(reflect.ValueOf(keyv))
	// 		// for li, lv := range block.Labels {
	// 		// 	fmt.Println(li, lv)
	// 		// 	lfieldIdx1 := blockTags.Labels[li].FieldIndex
	// 		// 	lfieldName1 := blockTags.Labels[li].Name
	// 		// 	if lfieldName1 == "___key" {
	// 		// 		continue
	// 		// 	}
		
				
	// 		// 	// if lfieldName == "___key" {
	// 		// 	// 	v.SetMapIndex(reflect.ValueOf(lv), reflect.ValueOf(v))
	// 		// 	// 	continue
	// 		// 	// }
		
	// 		// 	realv.Field(lfieldIdx1).Set(reflect.ValueOf(lv))
		
	// 		// 	if ix, exists := blockTags.LabelRange[lfieldName1]; exists {
	// 		// 		realv.Field(ix).Set(reflect.ValueOf(block.LabelRanges[li]))
	// 		// 	}
	// 		// }
	// 		// return diags
	// 	}
	// }
	for li, lv := range block.Labels {
		lfieldIdx := blockTags.Labels[li].FieldIndex
		lfieldName := blockTags.Labels[li].Name

		
		// if lfieldName == "___key" {
		// 	v.SetMapIndex(reflect.ValueOf(lv), reflect.ValueOf(v))
		// 	continue
		// }

		v.Field(lfieldIdx).Set(reflect.ValueOf(lv))

		if ix, exists := blockTags.LabelRange[lfieldName]; exists {
			v.Field(ix).Set(reflect.ValueOf(block.LabelRanges[li]))
		}
	}

	if blockTags.TypeRange != nil {
		v.Field(*blockTags.TypeRange).Set(reflect.ValueOf(block.TypeRange))
	}

	if blockTags.DefRange != nil {
		v.Field(*blockTags.DefRange).Set(reflect.ValueOf(block.DefRange))
	}

	return diags
}

// DecodeExpression extracts the value of the given expression into the given
// value. This value must be something that gocty is able to decode into,
// since the final decoding is delegated to that package.
//
// The given EvalContext is used to resolve any variables or functions in
// expressions encountered while decoding. This may be nil to require only
// constant values, for simple applications that do not support variables or
// functions.
//
// The returned diagnostics should be inspected with its HasErrors method to
// determine if the populated value is valid and complete. If error diagnostics
// are returned then the given value may have been partially-populated but
// may still be accessed by a careful caller for static analysis and editor
// integration use-cases.
func DecodeExpression(expr hcl.Expression, ctx *hcl.EvalContext, val interface{}) hcl.Diagnostics {
	srcVal, diags := expr.Value(ctx)

	convTy, err := gocty.ImpliedType(val)
	if err != nil {
		panic(fmt.Sprintf("unsuitable DecodeExpression target: %s", err))
	}

	srcVal, err = convert.Convert(srcVal, convTy)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsuitable value type",
			Detail:   fmt.Sprintf("Unsuitable value: %s", err.Error()),
			Subject:  expr.StartRange().Ptr(),
			Context:  expr.Range().Ptr(),
		})
		return diags
	}

	err = gocty.FromCtyValue(srcVal, val)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsuitable value type",
			Detail:   fmt.Sprintf("Unsuitable value: %s", err.Error()),
			Subject:  expr.StartRange().Ptr(),
			Context:  expr.Range().Ptr(),
		})
	}

	return diags
}
