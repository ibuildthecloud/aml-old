package eval

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

var _ ArrayValue = (*Array)(nil)

type Array struct {
	Position ast.Position
	Scope    *Scope

	array  *ast.Array
	values []Value
}

func (a *Array) Type(ctx context.Context) (Type, error) {
	return TypeArray, nil
}

func (a *Array) Empty(ctx context.Context) (bool, error) {
	if err := a.process(ctx); err != nil {
		return false, err
	}
	return len(a.values) == 0, nil
}

func (a *Array) Iterator(ctx context.Context) (Iterator, error) {
	if err := a.process(ctx); err != nil {
		return nil, err
	}
	return &iter{
		values: a.values,
		ctx:    ctx,
	}, nil
}

func (a *Array) Index(ctx context.Context, val Value) (Value, bool, error) {
	if t, err := val.Type(ctx); err != nil {
		return nil, false, err
	} else if t != TypeNumber {
		return nil, false, fmt.Errorf("can not use type %s as an index to an array", t)
	}
	obj, err := val.Interface(ctx)
	if err != nil {
		return nil, false, err
	}

	lookup, err := obj.(json.Number).Int64()
	if err != nil {
		return nil, false, fmt.Errorf("can only use valid integers as an index to an array, got %s: %w", obj, err)
	}

	iter, err := a.Iterator(ctx)
	if err != nil {
		return nil, false, err
	}

	for i := int64(0); ; i++ {
		v, cont, err := iter.Next()
		if err != nil {
			return nil, false, err
		}
		if !cont {
			break
		}

		if lookup == i {
			return v, true, nil
		}
	}

	return nil, false, nil
}

func (a *Array) Lookup(ctx context.Context, key string) (Value, bool, error) {
	//TODO implement me
	panic("implement me")
}

func (a *Array) process(ctx context.Context) error {
	if a.values != nil {
		return nil
	}

	var result []Value
	for i := range a.array.Values {
		val, err := ToValue(ctx, a.Scope, a.array.Values[i])
		if err != nil {
			return err
		}
		result = append(result, val)
	}

	a.values = result
	return nil
}

func (a *Array) Interface(ctx context.Context) (_ interface{}, err error) {
	defer func() {
		err = wrapErr(a.Position, err)
	}()
	if err := a.process(ctx); err != nil {
		return nil, err
	}

	// we don't want a nil array
	result := make([]interface{}, 0, len(a.values))
	for _, val := range a.values {
		v, err := val.Interface(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

type iter struct {
	index  int
	values []Value
	ctx    context.Context
}

func (i *iter) Next() (Value, bool, error) {
	tick(i.ctx)
	if i.index < len(i.values) {
		v := i.values[i.index]
		i.index++
		return v, true, nil
	}
	return nil, false, nil
}
