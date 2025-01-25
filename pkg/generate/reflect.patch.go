package generate

import (
	"fmt"
	"reflect"
)

// https://stackoverflow.com/questions/64196547/is-possible-to-reflect-an-struct-from-ast
func (g *Generator) ToReflectableStruct() (reflect.Value, error) {
	rootFields := []reflect.StructField{}
	for name, field := range g.Structs {
		thisStructFields := []reflect.StructField{}
		for _, field := range field.Fields {
			thisStructFields = append(thisStructFields, reflect.StructField{
				Name: field.Name,
				Type: reflect.TypeOf(field.Type),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, field.Name, field.Name)),
			})
		}
		rootFields = append(rootFields, reflect.StructField{
			Name: name,
			Type: reflect.StructOf(thisStructFields),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" hcl:"%s"`, name, name)),
		})
	}

	return reflect.New(reflect.StructOf(rootFields)), nil
}
