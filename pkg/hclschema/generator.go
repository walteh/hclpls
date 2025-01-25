package hclschema

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/walteh/hclpls/pkg/generate"
)

type flagConfig struct {
	Block bool
	Label bool
}

func RecursiveReflectableType(g *generate.Generator, name string, additionalFlags ...reflect.StructField) (reflect.Type, flagConfig, error) {
	flags := flagConfig{}
	structz, ok := g.Structs[name]
	if !ok {
		switch name {
		case "string":
			return reflect.TypeOf(""), flags, nil
		case "number":
			return reflect.TypeOf(0), flags, nil
		case "boolean", "bool":
			return reflect.TypeOf(false), flags, nil
		case "array":
			return reflect.TypeOf([]interface{}{}), flags, nil
		case "object":
			return reflect.TypeOf(map[string]interface{}{}), flags, nil
		case "any":
			return reflect.TypeOf(nil), flags, nil
		case "float":
			return reflect.TypeOf(float64(0)), flags, nil
		case "float32":
			return reflect.TypeOf(float32(0)), flags, nil
		case "float64":
			return reflect.TypeOf(float64(0)), flags, nil
		case "complex64":
			return reflect.TypeOf(complex64(0)), flags, nil
		case "complex128":
			return reflect.TypeOf(complex128(0)), flags, nil
		case "uint":
			return reflect.TypeOf(uint(0)), flags, nil
		case "uint8":
			return reflect.TypeOf(uint8(0)), flags, nil
		case "uint16":
			return reflect.TypeOf(uint16(0)), flags, nil
		case "uint32":
			return reflect.TypeOf(uint32(0)), flags, nil
		case "uint64":
			return reflect.TypeOf(uint64(0)), flags, nil
		case "int":
			return reflect.TypeOf(int(0)), flags, nil
		case "int32":
			return reflect.TypeOf(int32(0)), flags, nil
		case "int16":
			return reflect.TypeOf(int16(0)), flags, nil
		case "int8":
			return reflect.TypeOf(int8(0)), flags, nil
		default:
			if strings.HasPrefix(name, "*") {
				okay, sf, err := RecursiveReflectableType(g, name[1:])
				if err != nil {
					return nil, sf, fmt.Errorf("failed to get type for %s: %w", name, err)
				}
				return reflect.PointerTo(okay), sf, nil
			} else if strings.HasPrefix(name, "[]") {
				okay, _, err := RecursiveReflectableType(g, name[2:])
				if err != nil {
					return nil, flags, fmt.Errorf("failed to get type for %s: %w", name, err)
				}
				return reflect.SliceOf(okay), flags, nil
			} else if strings.HasPrefix(name, "map[") {
				before, after, ok := strings.Cut(name, "]")
				if !ok {
					return nil, flags, fmt.Errorf("failed to get type for %s", name)
				}
				before = strings.TrimPrefix(before, "map[")
				beforeType, _, err := RecursiveReflectableType(g, before)
				if err != nil {
					return nil, flags, fmt.Errorf("failed to get type for %s: %w", name, err)
				}
				afterType, _, err := RecursiveReflectableType(g, after, []reflect.StructField{
					{
						Name: "key",
						Type: beforeType,
						Tag:  reflect.StructTag(`json:"key" hcl:"key,label"`),
					},
				}...)
				if err != nil {
					return nil, flags, fmt.Errorf("failed to get type for %s: %w", name, err)
				}
				// flags.Block = true

				// copyOfAfterType := afterType

				return reflect.MapOf(beforeType, afterType), flags, nil
			}
			return nil, flags, fmt.Errorf("unknown type %q", name)
		}
	}

	fields := []reflect.StructField{}
	fields = append(fields, additionalFlags...)
	for _, field := range structz.Fields {
		fieldType, sf, err := RecursiveReflectableType(g, field.Type)
		if err != nil {
			return nil, sf, fmt.Errorf("failed to get type for field %s: %w", field.Name, err)
		}

		tag := reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, sf.hclTagForField(field)))

		fields = append(fields, reflect.StructField{
			Name: field.Name,
			Type: fieldType,
			Tag:  tag,
		})
	}

	slices.SortFunc(fields, func(a, b reflect.StructField) int {
		return strings.Compare(a.Name, b.Name)
	})
	flags.Block = true
	return reflect.StructOf(fields), flags, nil
}

func (f flagConfig) hclTagForField(field generate.Field) string {
	hcltags := []string{field.JSONName}
	if !field.Required {
		hcltags = append(hcltags, "optional")
	}
	if f.Block {
		hcltags = append(hcltags, "block")
	}
	if f.Label {
		hcltags = append(hcltags, "label")
	}
	return strings.Join(hcltags, ",")
}

// https://stackoverflow.com/questions/64196547/is-possible-to-reflect-an-struct-from-ast
func ToReflectableStruct(g *generate.Generator) (reflect.Type, error) {
	rootFields := []reflect.StructField{}

	rootStruct, ok := g.Structs["Root"]
	if ok {
		for _, field := range rootStruct.Fields {
			rrt, sf, err := RecursiveReflectableType(g, field.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to get type for field %s: %w", field.Name, err)
			}
			tag := reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, sf.hclTagForField(field)))
			rootFields = append(rootFields, reflect.StructField{Name: field.Name, Type: rrt, Tag: tag})
		}
	}

	// // we want the root to take over and return a nested bunch of nested structs
	// for name, field := range g.Structs {
	// 	if name == "Root" {
	// 		continue
	// 	}
	// 	thisStructFields := []reflect.StructField{}
	// 	for _, field := range field.Fields {
	// 		rrt, err := RecursiveReflectableType(g, field.Type)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to get type for field %s: %w", field.Name, err)
	// 		}
	// 		tag := reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, field.JSONName))
	// 		thisStructFields = append(thisStructFields, reflect.StructField{Name: field.Name, Type: rrt, Tag: tag})
	// 	}
	// 	rootFields = append(rootFields, reflect.StructField{
	// 		Name: name,
	// 			Type: reflect.StructOf(thisStructFields),
	// 		Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, name, name)),
	// 	})
	// }

	slices.SortFunc(rootFields, func(a, b reflect.StructField) int {
		return strings.Compare(a.Name, b.Name)
	})

	return reflect.StructOf(rootFields), nil
}
