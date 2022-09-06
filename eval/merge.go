package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type ValuePosition struct {
	Value    Value
	Position ast.Position
}

func MergeSlice(ctx context.Context, scope *Scope, objs ...ValuePosition) (_ Value, err error) {
	if len(objs) == 0 {
		return nil, nil
	}

	result := objs[0].Value
	for i := 1; i < len(objs); i++ {
		result, err = Merge(ctx, objs[i].Position, result, objs[i].Value)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func Merge(ctx context.Context, pos ast.Position, left, right Value) (_ Value, err error) {
	return merge(ctx, nil, pos, "&", left, right)
}

func merge(ctx context.Context, _ *Scope, pos ast.Position, _ string, left, right Value) (_ Value, err error) {
	defer func() {
		err = wrapErr(pos, err)
	}()

	if left == nil {
		return right, nil
	}

	if right == nil {
		return left, nil
	}

	lvt, err := left.Type(ctx)
	if err != nil {
		return nil, err
	}

	rvt, err := right.Type(ctx)
	if err != nil {
		return nil, err
	}

	if lvt == TypeNull || rvt == TypeNull {
		return right, nil
	}

	if rvt == TypeNull {
		return right, nil
	}

	if lvt != rvt {
		return nil, fmt.Errorf("can not merge incompatible types %s and %s", lvt, rvt)
	}

	if lvt != TypeObject {
		return right, nil
	}

	leftObject := left.(ObjectValue)
	rightObject := right.(ObjectValue)
	return MergeObjects(ctx, leftObject, rightObject)
}
