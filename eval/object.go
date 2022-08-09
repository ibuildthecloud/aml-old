package eval

import (
	"context"

	"github.com/acorn-io/aml/parser/ast"
)

type ObjectReference struct {
	Position ast.Position
	Scope    *Scope

	object *ast.Object
}

func (o *ObjectReference) Lookup(key string) (Reference, error) {
	//TODO implement me
	panic("implement me")
}

func (o *ObjectReference) Resolve(ctx context.Context) (Value, error) {
	var (
		keyNames = map[string]bool{}
		keyOrder []string
		fields   = map[string]*FieldReference{}
	)

	for i := range o.object.Fields {
		fr := FieldReference{
			Field: &o.object.Fields[i],
		}
		for _, key := range fr.Keys() {
			if keyNames[key] {
				continue
			}
			keyOrder = append(keyOrder, key)
			cp := fr
			cp.Key = key
			fields[key] = &cp
		}
	}

	data := map[string]interface{}{}

	for _, key := range keyOrder {
		v, err := fields[key].Resolve(ctx)
		if err != nil {
			return nil, err
		}
		data[key] = v.Interface()
	}

	return &Object{data: data}, nil
}

type FieldReference struct {
	Scope *Scope
	Key   string
	Field *ast.Field
	ref   Reference
}

func (f *FieldReference) Resolve(ctx context.Context) (Value, error) {
	if f.ref != nil {
		return f.ref.Resolve(ctx)
	}
	if f.Field.Value != nil {
		f.ref = toReference(f.Scope, f.Field.Value)
	}
	return f.ref.Resolve(ctx)
}

func (f *FieldReference) Keys() []string {
	return []string{f.Field.Key}
}
