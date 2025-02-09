package hclschema

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/walteh/hclpls/pkg/generate"
)

type flagConfig struct {
	Block        bool
	KeyBlockType reflect.Type
}

const mapKeyName = "___key"

type AnyType struct{}

var anyType = reflect.TypeOf(AnyType{})

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
			return reflect.TypeOf([]any{}), flags, nil
		case "object":
			return reflect.TypeOf(map[string]any{}), flags, nil
		case "any", "interface{}":
			return reflect.TypeFor[any](), flags, nil
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
				nme := name[2:]
				okay, _, err := RecursiveReflectableType(g, nme)
				if err != nil {
					return nil, flags, fmt.Errorf("failed to get type for %s: %w", name, err)
				}
				if _, ok := g.Structs[nme]; ok {
					flags.Block = true
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

				afterType, _, err := RecursiveReflectableType(g, after)
				if err != nil {
					return nil, flags, fmt.Errorf("failed to get type for %s: %w", name, err)
				}

				if afterType.Kind() == reflect.Pointer {
					after = after[1:]
				}

				if _, ok := g.Structs[after]; ok && beforeType.Kind() == reflect.String {
					flags.Block = true
				}
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
	// if f.Label {
	// 	hcltags = append(hcltags, "label")
	// }
	return strings.Join(hcltags, ",")
}

// https://stackoverflow.com/questions/64196547/is-possible-to-reflect-an-struct-from-ast
func ToReflectableStruct(g *generate.Generator) (reflect.Type, error) {
	rootFields := []reflect.StructField{}

	var rootStruct *generate.Struct
	for _, field := range g.Structs {
		if f, ok := field.Fields["Schema"]; ok && f.JSONName == "$schema" {
			rootStruct = &field
			break
		} else if field.Name == "Root" {
			rootStruct = &field
		}
	}
	if rootStruct == nil {
		return nil, fmt.Errorf("no root struct found")
	}

	for _, field := range rootStruct.Fields {
		if field.Name == "Schema" {
			continue
		}
		rrt, sf, err := RecursiveReflectableType(g, field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get type for field %s: %w", field.Name, err)
		}

		tagString := sf.hclTagForField(field)
		tag := reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, tagString))
		rootFields = append(rootFields, reflect.StructField{Name: field.Name, Type: rrt, Tag: tag})

		// if sf.KeyBlockType != nil {
		// 	tag2 := reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, sf.hclTagForField(field)))
		// 	rrt = reflect.SliceOf(sf.KeyBlockType)
		// 	rootFields = append(rootFields, reflect.StructField{Name: field.JSONName, Type: rrt, Tag: tag2})
		// }

	}

	slices.SortFunc(rootFields, func(a, b reflect.StructField) int {
		return strings.Compare(a.Name, b.Name)
	})

	return reflect.StructOf(rootFields), nil
}
