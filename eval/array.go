package eval

import (
	"context"

	"github.com/acorn-io/aml/parser/ast"
)

type ArrayValue struct {
	Position ast.Position
	Value    []interface{}
}

func (a *ArrayValue) Interface() interface{} {
	return a.Value
}

type ArrayReference struct {
	Position ast.Position
	Scope    *Scope

	array *ast.Array
	ref   []Reference
	val   Value
}

func (a *ArrayReference) Type() (Type, error) {
	return TypeArray, nil
}

func (a *ArrayReference) Lookup(key string) (Reference, error) {
	//TODO implement me
	panic("implement me")
}

func (a *ArrayReference) Resolve(ctx context.Context) (Value, error) {
	if a.ref == nil {
		for i := range a.array.Values {
			a.ref = append(a.ref, toReference(a.Scope, a.array.Values[i]))
		}
	}

	if a.val == nil {
		// we don't want a nil array
		result := make([]interface{}, 0, 0)
		for _, ref := range a.ref {
			v, err := ref.Resolve(ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, v.Interface())
		}
		a.val = &ArrayValue{Value: result}
	}

	return a.val, nil
}
