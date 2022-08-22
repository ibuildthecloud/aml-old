package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

const (
	ReturnName = "_return"
)

var (
	_        ObjectValue = (*ObjectReference)(nil)
	ArgsName             = "_args"
)

type ObjectReference struct {
	Position ast.Position
	Scope    *Scope
	Fields   []*ast.Field
	Values   map[string]Value

	fields         []*FieldReference
	keyOrder       []string
	embedded       *bool
	embeddedValue  Value
	embeddedLookup bool
}

func EmptyObjectReference(pos ast.Position, scope *Scope) *ObjectReference {
	return &ObjectReference{
		Position: pos,
		Scope:    scope,
	}
}

func (o *ObjectReference) Keys(ctx context.Context) (result []string, _ error) {
	o.process()
	if o.keyOrder != nil {
		return o.keyOrder, nil
	}

	keyNames := map[string]bool{}
	for _, f := range o.fields {
		if f.Field.Let {
			continue
		}
		keys, err := f.Keys(ctx)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if keyNames[key] {
				continue
			}
			keyNames[key] = true
			result = append(result, key)
		}
	}

	o.keyOrder = result
	return o.keyOrder, nil
}

func (o *ObjectReference) getEmbeddedObject(ctx context.Context) (Value, error) {
	if o.embeddedValue != nil {
		return o.embeddedValue, nil
	}

	v, _, err := o.Lookup(ctx, EmbeddedKey)
	if err != nil {
		return nil, err
	}
	o.embeddedValue = v
	return v, nil
}

func (o *ObjectReference) Type(ctx context.Context) (Type, error) {
	if ok, err := o.isEmbedded(); err != nil {
		return TypeObject, err
	} else if ok {
		if o, err := o.getEmbeddedObject(ctx); err != nil {
			return TypeObject, err
		} else {
			return o.Type(ctx)
		}
	}
	return TypeObject, nil
}

func (o *ObjectReference) Call(ctx context.Context, scope *Scope, args *ast.Value) (_ Value, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	call := &ObjectReference{
		Position: o.Position,
		Scope:    scope,
		Fields: []*ast.Field{
			{
				Position: args.Position,
				Key: ast.Key{
					Position: args.Position,
					Name: &ast.String{
						Position: args.Position,
						Parts: []ast.StringPart{
							{
								String: &ArgsName,
							},
						},
					},
				},
				Value: args,
			},
		},
	}

	obj, err := o.Merge(ctx, call)
	if err != nil {
		return nil, fmt.Errorf("error in call merge: %w", err)
	}
	ret, ok, err := obj.Lookup(ctx, ReturnName)
	if !ok {
		return nil, fmt.Errorf("invalid function missing return key")
	}
	if err != nil {
		return nil, fmt.Errorf("err in call return: %w", err)
	}
	return ret, err
}

func (o *ObjectReference) isEmbedded() (bool, error) {
	if o.embedded != nil {
		return *o.embedded, nil
	}

	o.process()

	if len(o.fields) == 0 {
		return false, nil
	}

	embedded, err := o.fields[0].Embedded()
	if err != nil {
		return false, err
	}

	for i := 1; i < len(o.fields); i++ {
		newEmbedded, err := o.fields[i].Embedded()
		if err != nil {
			return false, err
		}
		if newEmbedded != embedded {
			return false, fmt.Errorf("can not mix embedded objects with fields")
		}
	}

	o.embedded = &embedded
	return embedded, nil
}

func (o *ObjectReference) Lookup(ctx context.Context, key string) (_ Value, _ bool, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	if key != EmbeddedKey {
		if ok, err := o.isEmbedded(); err != nil {
			return nil, false, err
		} else if ok {
			if o.embeddedLookup {
				return nil, false, nil
			}
			o.embeddedLookup = true
			defer func() {
				o.embeddedLookup = false
			}()
			if o, err := o.getEmbeddedObject(ctx); err != nil {
				return nil, false, err
			} else {
				return o.Lookup(ctx, key)
			}
		}
	}

	o.process()

	if v, ok := o.Values[key]; ok {
		return v, true, nil
	}

	var (
		values []ValuePosition
	)
	for _, f := range o.fields {
		v, ok, err := f.Value(ctx, key)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			continue
		}
		values = append(values, ValuePosition{
			Value:    v,
			Position: f.Field.Position,
		})
	}

	if len(values) == 0 {
		return nil, false, nil
	}

	v, err := MergeSlice(ctx, o.Scope, values...)
	if err != nil {
		return nil, false, err
	}

	if o.Values == nil {
		o.Values = map[string]Value{}
	}
	o.Values[key] = v
	return v, true, nil
}

func (o *ObjectReference) process() {
	if o.fields != nil {
		return
	}

	var (
		parent = &ObjectReference{
			Position: o.Position,
			Scope:    o.Scope,
		}
		fieldList []*FieldReference
	)

	for _, field := range o.Fields {
		if field.For != nil || field.If != nil {
			fieldList = append(fieldList, &FieldReference{
				Field: field,
				Scope: o.Scope.Push(parent),
			})
		} else {
			parent.fields = append(parent.fields, &FieldReference{
				Field: field,
				Scope: o.Scope.Push(parent),
			})
			fieldList = append(fieldList, &FieldReference{
				Field: field,
				Scope: o.Scope.Push(o),
			})
		}
	}

	o.fields = fieldList
}

func (o *ObjectReference) Interface(ctx context.Context) (_ interface{}, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	if ok, err := o.isEmbedded(); err != nil {
		return nil, err
	} else if ok {
		if o, err := o.getEmbeddedObject(ctx); err != nil {
			return nil, err
		} else {
			return o.Interface(ctx)
		}
	}

	o.process()

	data := map[string]interface{}{}

	values, err := o.getValues(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range values {
		data[v.Key], err = v.Value.Interface(ctx)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (o *ObjectReference) getValues(ctx context.Context) (_ []Local, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	keys, err := o.Keys(ctx)
	if err != nil {
		return nil, err
	}
	var result []Local
	for _, key := range keys {
		v, ok, err := o.Lookup(ctx, key)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		result = append(result, Local{
			Key:   key,
			Value: v,
		})
	}
	return result, nil
}

func (o *ObjectReference) GetPosition(ctx context.Context) (ast.Position, error) {
	return o.Position, nil
}

func (o *ObjectReference) GetScope(ctx context.Context) (*Scope, error) {
	return o.Scope, nil
}

func (o *ObjectReference) GetFields(ctx context.Context) ([]*ast.Field, error) {
	return o.Fields, nil
}

func (o *ObjectReference) Index(ctx context.Context, val Value) (Value, bool, error) {
	if t, err := val.Type(ctx); err != nil {
		return nil, false, err
	} else if t != TypeString {
		return nil, false, fmt.Errorf("can not use type %s as an index to an object", t)
	}
	obj, err := val.Interface(ctx)
	if err != nil {
		return nil, false, err
	}

	return o.Lookup(ctx, obj.(string))
}

func (o *ObjectReference) Merge(ctx context.Context, val ObjectValue) (Value, error) {
	if ok, err := o.isEmbedded(); err != nil {
		return nil, err
	} else if ok {
		if o, err := o.getEmbeddedObject(ctx); err != nil {
			return nil, err
		} else {
			return o.(ObjectValue).Merge(ctx, val)
		}
	}

	o.process()

	rightPosition, err := val.GetPosition(ctx)
	if err != nil {
		return nil, err
	}

	rightFields, err := val.GetFields(ctx)
	if err != nil {
		return nil, err
	}

	rightScope, err := val.GetScope(ctx)
	if err != nil {
		return nil, err
	}

	return &ObjectReference{
		Position: rightPosition,
		Scope:    o.Scope.Merge(rightScope),
		Fields:   append(o.Fields, rightFields...),
	}, nil
}
