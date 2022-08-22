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

	values   map[string]Value
	fields   []*FieldReference
	keyOrder []string
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

func (o *ObjectReference) Type(ctx context.Context) (_ Type, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()

	keys, err := o.Keys(ctx)
	if err != nil {
		return TypeObject, err
	}
	for _, key := range keys {
		if key == EmbeddedKey {
			v, ok, err := o.Lookup(ctx, key)
			if err != nil {
				return TypeObject, err
			}
			if !ok {
				return TypeObject, fmt.Errorf("failed to determine type of object")
			}
			return v.Type(ctx)
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

func (o *ObjectReference) Lookup(ctx context.Context, key string) (_ Value, _ bool, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	o.process()

	if v, ok := o.values[key]; ok {
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

	if o.values == nil {
		o.values = map[string]Value{}
	}
	o.values[key] = v
	return v, true, nil
}

func (o *ObjectReference) process() {
	if o.fields != nil {
		return
	}

	var fieldList []*FieldReference
	for _, field := range o.Fields {
		fieldList = append(fieldList, &FieldReference{
			Field: field,
			Scope: o.Scope.Push(o),
		})
	}

	o.fields = fieldList
}

func (o *ObjectReference) Interface(ctx context.Context) (_ interface{}, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

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

	if v, ok := data[EmbeddedKey]; ok {
		if len(data) > 1 {
			return nil, fmt.Errorf("object evaluated to mixture of object and embedded value")
		}
		return v, nil
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
