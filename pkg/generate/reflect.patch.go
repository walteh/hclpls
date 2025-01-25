package generate

// func (g *Generator) RecursiveReflectableType(name string) (reflect.Type, error) {
// 	structz, ok := g.Structs[name]
// 	if !ok {
// 		switch name {
// 		case "string":
// 			return reflect.TypeOf(""), nil
// 		case "number":
// 			return reflect.TypeOf(0), nil
// 		case "boolean":
// 			return reflect.TypeOf(false), nil
// 		case "array":
// 			return reflect.TypeOf([]interface{}{}), nil
// 		case "object":
// 			return reflect.TypeOf(map[string]interface{}{}), nil
// 		case "any":
// 			return reflect.TypeOf(nil), nil
// 		default:
// 			if strings.HasPrefix(name, "*") {
// 				okay, err := g.RecursiveReflectableType(name[1:])
// 				if err != nil {
// 					return nil, fmt.Errorf("failed to get type for %s: %w", name, err)
// 				}
// 				return reflect.PointerTo(okay), nil
// 			} else if strings.HasPrefix(name, "[]") {
// 				okay, err := g.RecursiveReflectableType(name[2:])
// 				if err != nil {
// 					return nil, fmt.Errorf("failed to get type for %s: %w", name, err)
// 				}
// 				return reflect.SliceOf(okay), nil
// 			} else if strings.HasPrefix(name, "map[") {
// 				before, after, ok := strings.Cut(name, "]")
// 				if !ok {
// 					return nil, fmt.Errorf("failed to get type for %s", name)
// 				}
// 				before = strings.TrimPrefix(before, "map[")
// 				beforeType, err := g.RecursiveReflectableType(before)
// 				if err != nil {
// 					return nil, fmt.Errorf("failed to get type for %s: %w", name, err)
// 				}
// 				afterType, err := g.RecursiveReflectableType(after)
// 				if err != nil {
// 					return nil, fmt.Errorf("failed to get type for %s: %w", name, err)
// 				}
// 				return reflect.MapOf(beforeType, afterType), nil
// 			}
// 			return nil, fmt.Errorf("unknown type %s", name)
// 		}
// 	}

// 	fields := []reflect.StructField{}
// 	for _, field := range structz.Fields {
// 		fieldType, err := g.RecursiveReflectableType(field.Type)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to get type for field %s: %w", field.Name, err)
// 		}
// 		fields = append(fields, reflect.StructField{
// 			Name: field.Name,
// 			Type: fieldType,
// 			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, field.JSONName)),
// 		})
// 	}
// 	return reflect.StructOf(fields), nil
// }

// // https://stackoverflow.com/questions/64196547/is-possible-to-reflect-an-struct-from-ast
// func (g *Generator) ToReflectableStruct() (reflect.Type, error) {
// 	rootFields := []reflect.StructField{}
// 	// we want the root to take over and return a nested bunch of nested structs
// 	for name, field := range g.Structs {
// 		thisStructFields := []reflect.StructField{}
// 		for _, field := range field.Fields {
// 			rrt, err := g.RecursiveReflectableType(field.Type)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to get type for field %s: %w", field.Name, err)
// 			}
// 			tag := reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.JSONName, field.JSONName))
// 			if name == "Root" {
// 				rootFields = append(rootFields, reflect.StructField{Name: field.Name, Type: rrt, Tag: tag})
// 			} else {
// 				thisStructFields = append(thisStructFields, reflect.StructField{Name: field.Name, Type: rrt, Tag: tag})
// 			}
// 		}
// 		if name != "Root" {
// 			rootFields = append(rootFields, reflect.StructField{
// 				Name: name,
// 				Type: reflect.StructOf(thisStructFields),
// 				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, name, name)),
// 			})
// 		}
// 	}

// 	return reflect.StructOf(rootFields), nil
// }
