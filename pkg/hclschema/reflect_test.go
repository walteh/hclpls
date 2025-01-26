package hclschema_test

import "reflect"

// Convert named types to anonymous structs
func toAnonymousType(v interface{}) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		return t
	}

	fields := make([]reflect.StructField, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		newField := reflect.StructField{
			Name: f.Name,
			Type: toAnonymousFieldType(f.Type),
			Tag:  f.Tag,
		}
		fields[i] = newField
	}

	return reflect.StructOf(fields)
}

// Convert named values to anonymous values
func toAnonymousValue(v interface{}) interface{} {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	t := val.Type()
	if t.Kind() != reflect.Struct {
		return v
	}

	// Create new anonymous struct with same fields
	newType := toAnonymousType(val.Interface())
	newVal := reflect.New(newType).Elem()

	// Copy fields, converting nested structs recursively
	for i := 0; i < t.NumField(); i++ {
		field := val.Field(i)
		if !field.IsValid() {
			continue
		}

		newField := newVal.Field(i)
		switch field.Kind() {
		case reflect.Ptr:
			if field.IsNil() {
				continue
			}
			converted := toAnonymousValue(field.Interface())
			if converted != nil {
				newField.Set(reflect.ValueOf(converted))
			}
		case reflect.Slice:
			if field.IsNil() {
				continue
			}
			newSlice := reflect.MakeSlice(field.Type(), field.Len(), field.Cap())
			for j := 0; j < field.Len(); j++ {
				elem := toAnonymousValue(field.Index(j).Interface())
				if elem != nil {
					newSlice.Index(j).Set(reflect.ValueOf(elem))
				}
			}
			newField.Set(newSlice)
		case reflect.Map:
			if field.IsNil() {
				continue
			}
			newMap := reflect.MakeMap(field.Type())
			iter := field.MapRange()
			for iter.Next() {
				k := iter.Key()
				v := toAnonymousValue(iter.Value().Interface())
				if v != nil {
					newMap.SetMapIndex(k, reflect.ValueOf(v))
				}
			}
			newField.Set(newMap)
		case reflect.Struct:
			converted := toAnonymousValue(field.Interface())
			if converted != nil {
				newField.Set(reflect.ValueOf(converted))
			}
		default:
			newField.Set(field)
		}
	}

	// If original was a pointer, return pointer
	if v != nil && reflect.TypeOf(v).Kind() == reflect.Ptr {
		ptr := reflect.New(newType)
		ptr.Elem().Set(newVal)
		return ptr.Interface()
	}

	return newVal.Interface()
}

func toAnonymousFieldType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr:
		return reflect.PointerTo(toAnonymousFieldType(t.Elem()))
	case reflect.Slice:
		return reflect.SliceOf(toAnonymousFieldType(t.Elem()))
	case reflect.Map:
		return reflect.MapOf(t.Key(), toAnonymousFieldType(t.Elem()))
	case reflect.Struct:
		fields := make([]reflect.StructField, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			newField := reflect.StructField{
				Name: f.Name,
				Type: toAnonymousFieldType(f.Type),
				Tag:  f.Tag,
			}
			fields[i] = newField
		}
		return reflect.StructOf(fields)
	default:
		return t
	}
}
