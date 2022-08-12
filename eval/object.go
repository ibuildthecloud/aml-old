package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type ObjectReference struct {
	Position ast.Position
	Scope    *Scope

	object    *ast.Object
	keyOrder  []string
	fields    map[string]*FieldReference
	resolving bool
}

func (o *ObjectReference) Type() (Type, error) {
	return TypeObject, nil
}

func (o *ObjectReference) Lookup(key string) (Reference, error) {
	o.processFields()
	f, ok := o.fields[key]
	if !ok {
		return nil, &ErrKeyNotFound{key}
	}
	return f, nil
}

func (o *ObjectReference) processFields() {
	if o.fields != nil {
		return
	}

	var (
		keyNames = map[string]bool{}
		keyOrder []string
		fields   = map[string]*FieldReference{}
	)

	for i := range o.object.Fields {
		fr := FieldReference{
			Field: o.object.Fields[i],
			Scope: o.Scope.Push(o),
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

	o.keyOrder = keyOrder
	o.fields = fields
	return
}

func (o *ObjectReference) Resolve(ctx context.Context) (_ Value, err error) {
	defer func() {
		o.resolving = false
		err = wrapErr(o.Position, err)
	}()
	if o.resolving {
		return nil, fmt.Errorf("cycle detected")
	}
	o.resolving = true
	o.processFields()

	data := map[string]interface{}{}

	for _, key := range o.keyOrder {
		v, err := o.fields[key].Resolve(ctx)
		if err != nil {
			return nil, err
		}
		data[key] = v.Interface()
	}

	return &Object{data: data}, nil
}

type FieldReference struct {
	Scope     *Scope
	Key       string
	Field     *ast.Field
	ref       Reference
	resolving bool
}

func (f *FieldReference) Type() (Type, error) {
	f.process()
	return f.ref.Type()
}

func (f *FieldReference) process() {
	if f.ref != nil {
		return
	}
	f.ref = toReference(f.Scope, f.Field.Value)
}

func (f *FieldReference) Lookup(key string) (_ Reference, err error) {
	defer func() {
		err = wrapErr(f.Field.Position, err)
	}()
	f.process()
	return f.ref.Lookup(key)
}

func (f *FieldReference) Resolve(ctx context.Context) (_ Value, err error) {
	defer func() {
		f.resolving = false
		err = wrapErr(f.Field.Position, err)
	}()
	if f.resolving {
		return nil, fmt.Errorf("cycle detected")
	}
	f.resolving = true

	f.process()
	return f.ref.Resolve(ctx)
}

func (f *FieldReference) Keys() []string {
	return []string{f.Field.Key}
}
