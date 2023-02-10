package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

const (
	ReturnName = "return"
)

var (
	_        ObjectValue = (*ObjectReference)(nil)
	_        Callable    = (*ObjectReference)(nil)
	ArgsName             = "args"
)

type ObjectReference struct {
	Position ast.Position
	Scope    *Scope
	Fields   []*ast.Field

	values   map[string]Value
	fields   []*FieldReference
	keyOrder []string
}

func (o *ObjectReference) GetPosition() ast.Position {
	return o.Position
}

func (o *ObjectReference) Len(ctx context.Context) (int, error) {
	l, err := o.Keys(ctx)
	if err != nil {
		return 0, err
	}
	if len(l) == 1 && l[0] == EmbeddedKey {
		return 0, nil
	}
	return len(l), nil
}

func (o *ObjectReference) Keys(ctx context.Context) (result []string, _ error) {
	o.process()
	if o.keyOrder != nil {
		return o.keyOrder, nil
	}

	keyNames := map[string]bool{}
	for _, f := range o.fields {
		if f.Field.Let != nil {
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

func (o *ObjectReference) Slice(ctx context.Context, start, end Value) (_ Value, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	return nil, fmt.Errorf("can not slice an object")
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

	values, err := o.KeyValues(ctx)
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

func (o *ObjectReference) KeyValues(ctx context.Context) (_ []KeyValue, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	keys, err := o.Keys(ctx)
	if err != nil {
		return nil, err
	}
	var result []KeyValue
	for _, key := range keys {
		v, ok, err := o.Lookup(ctx, key)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		result = append(result, KeyValue{
			Key:   key,
			Value: v,
		})
	}
	return result, nil
}

func (o *ObjectReference) GetScope(ctx context.Context) (*Scope, error) {
	return o.Scope, nil
}

func (o *ObjectReference) GetFields(ctx context.Context) ([]*ast.Field, error) {
	return o.Fields, nil
}

func (o *ObjectReference) Index(ctx context.Context, val Value) (Value, error) {
	if t, err := val.Type(ctx); err != nil {
		return nil, err
	} else if t != TypeString {
		return nil, fmt.Errorf("can not use type %s as an index to an object", t)
	}
	obj, err := val.Interface(ctx)
	if err != nil {
		return nil, err
	}
	v, ok, err := o.Lookup(ctx, obj.(string))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &ErrKeyNotFound{
			Key: obj.(string),
		}
	}
	return v, nil
}

func MergeObjects(ctx context.Context, left, right ObjectValue) (Value, error) {
	rightPosition := right.GetPosition()

	rightFields, err := right.GetFields(ctx)
	if err != nil {
		return nil, err
	}

	rightScope, err := right.GetScope(ctx)
	if err != nil {
		return nil, err
	}

	leftScope, err := left.GetScope(ctx)
	if err != nil {
		return nil, err
	}

	leftFields, err := left.GetFields(ctx)
	if err != nil {
		return nil, err
	}

	return &ObjectReference{
		Position: rightPosition,
		Scope:    leftScope.Merge(rightScope),
		Fields:   append(leftFields, rightFields...),
	}, nil
}
