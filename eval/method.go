package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type MethodFunc func(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error)

func (m MethodFunc) GetPosition() ast.Position {
	return ast.Position{}
}

func (m MethodFunc) Slice(ctx context.Context, start, end Value) (_ Value, err error) {
	return nil, fmt.Errorf("can not slice a method")
}

func (m MethodFunc) Type(ctx context.Context) (Type, error) {
	return TypeBuiltin, nil
}

func (m MethodFunc) Lookup(ctx context.Context, key string) (Value, bool, error) {
	return nil, false, nil
}

func (m MethodFunc) Index(ctx context.Context, val Value) (Value, error) {
	return nil, fmt.Errorf("invalid ")
}

func (m MethodFunc) Interface(ctx context.Context) (any, error) {
	return nil, nil
}

func (m MethodFunc) Call(ctx context.Context, scope *Scope, pos ast.Position, args []KeyValue) (Value, error) {
	var vals []Value
	for _, kv := range args {
		vals = append(vals, kv.Value)
	}
	return m(ctx, scope, pos, vals...)
}
