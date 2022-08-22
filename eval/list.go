package eval

import (
	"context"
	"fmt"
	"strconv"

	"github.com/acorn-io/aml/parser/ast"
)

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

	if t != TypeArray {
		return nil, fmt.Errorf("expression must evaluate to an array, got %s", t)
	}

	array := obj.(ArrayValue)
	iter, err := array.Iterator(ctx)
	if err != nil {
		return nil, err
	}

	var val []Value
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

		locals := &Locals{}
		locals.Add(expr.IndexVar, &Scalar{
			Position: expr.Position,
			Scope:    scope,
			Number:   &index,
		})
		locals.Add(expr.ValueVar, v)

		val = append(val, ToObject(scope.Push(locals), expr.Object))
	}

	return &Array{
		Position: expr.Position,
		Scope:    scope,
		values:   val,
	}, nil
}
