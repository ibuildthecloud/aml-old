package eval

import (
	"context"
	"fmt"
	"strconv"

	"github.com/acorn-io/aml/parser/ast"
)

func EvaluateArray(ctx context.Context, scope *Scope, array *ast.Array) (result []Value, err error) {
	v, err := ToValue(ctx, scope, &ast.Value{
		Position: array.Position,
		Array:    array,
	})
	if err != nil {
		return nil, err
	}
	iter, err := v.(ArrayValue).Iterator(ctx)
	if err != nil {
		return nil, err
	}
	for {
		v, cont, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !cont {
			break
		}
		result = append(result, v)
	}

	return result, nil
}

type item struct {
	Key   Value
	Value Value
}

func itemsForObject(ctx context.Context, pos ast.Position, obj ObjectValue) (result []item, _ error) {
	kvs, err := obj.KeyValues(ctx)
	if err != nil {
		return nil, err
	}

	for _, kv := range kvs {
		key := kv.Key
		result = append(result, item{
			Key: &Scalar{
				Position: pos,
				String:   &key,
			},
			Value: kv.Value,
		})
	}

	return
}

func itemsForArray(ctx context.Context, pos ast.Position, array ArrayValue) (result []item, _ error) {
	iter, err := array.Iterator(ctx)
	if err != nil {
		return nil, err
	}

	for i := 0; ; i++ {
		tick(ctx)
		v, cont, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !cont {
			break
		}

		index := ast.Number(strconv.Itoa(i))

		result = append(result, item{
			Key: &Scalar{
				Position: pos,
				Number:   &index,
			},
			Value: v,
		})
	}
	return
}

func EvaluateList(ctx context.Context, scope *Scope, expr *ast.For) (_ *Array, err error) {
	defer func() {
		err = wrapErr(expr.Position, err)
	}()
	obj, err := EvaluateExpression(ctx, scope, expr.Array)
	if err != nil {
		return nil, err
	}

	t, err := obj.Type(ctx)
	if err != nil {
		return nil, err
	}

	var items []item
	if t == TypeArray {
		items, err = itemsForArray(ctx, expr.Position, obj.(ArrayValue))
		if err != nil {
			return nil, err
		}
	} else if t == TypeObject {
		items, err = itemsForObject(ctx, expr.Position, obj.(ObjectValue))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("expression must evaluate to an array or object, got %s", t)
	}

	// we do not want this to be nil, so assign it to empty
	val := []Value{}
	for _, item := range items {
		tick(ctx)

		var last Value = &ObjectReference{
			Position: expr.Position,
			Scope:    &Scope{},
		}

		if len(val) > 0 {
			last = val[len(val)-1]
		}

		locals := &Locals{}
		locals.Add("last", last)
		locals.Add(expr.IndexVar, item.Key)
		locals.Add(expr.ValueVar, item.Value)

		if expr.Condition != nil {
			v, err := EvaluateExpression(ctx, scope.Push(locals), expr.Condition)
			if err != nil {
				return nil, err
			}
			b, err := expectBool(ctx, v)
			if err != nil {
				return nil, err
			}
			if !b {
				continue
			}
		}

		val = append(val, ToObject(scope.Push(locals), expr.Object))
	}

	return &Array{
		Position: expr.Position,
		Scope:    scope,
		Values:   val,
	}, nil
}
