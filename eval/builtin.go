package eval

import (
	"context"
	"fmt"
	"strconv"

	"github.com/acorn-io/aml/parser/ast"
)

type Builtin struct {
	Values map[string]Value
}

func NewBuiltin() *Builtin {
	return &Builtin{
		Values: map[string]Value{
			"len":   MethodFunc(length),
			"print": MethodFunc(printf),
		},
	}
}

func (b *Builtin) Slice(ctx context.Context, start, end Value) (_ Value, err error) {
	return nil, fmt.Errorf("can not slice a builtin")
}

func (b *Builtin) Type(ctx context.Context) (Type, error) {
	return TypeBuiltin, nil
}

func (b *Builtin) Lookup(ctx context.Context, key string) (Value, bool, error) {
	v, ok := b.Values[key]
	return v, ok, nil
}

func (b *Builtin) Index(ctx context.Context, val Value) (Value, bool, error) {
	s, err := val.Interface(ctx)
	if err != nil {
		return nil, false, err
	}
	if key, ok := s.(string); ok {
		return b.Lookup(ctx, key)
	}
	return nil, false, nil
}

func (b *Builtin) Interface(ctx context.Context) (any, error) {
	return nil, nil
}

type MethodFunc func(ctx context.Context, scope *Scope, pos ast.Position, args Value) (Value, error)

func (m MethodFunc) Slice(ctx context.Context, start, end Value) (_ Value, err error) {
	return nil, fmt.Errorf("can not slice a method")
}

func (m MethodFunc) Type(ctx context.Context) (Type, error) {
	return TypeBuiltin, nil
}

func (m MethodFunc) Lookup(ctx context.Context, key string) (Value, bool, error) {
	return nil, false, nil
}

func (m MethodFunc) Index(ctx context.Context, val Value) (Value, bool, error) {
	return nil, false, nil
}

func (m MethodFunc) Interface(ctx context.Context) (any, error) {
	return nil, nil
}

func (m MethodFunc) Call(ctx context.Context, scope *Scope, args *ast.Value) (Value, error) {
	v, err := ToValue(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	return m(ctx, scope, args.Position, v)
}

func printf(ctx context.Context, scope *Scope, pos ast.Position, args Value) (Value, error) {
	print(args.Interface(ctx))
	println()
	return &Scalar{
		Position: pos,
		Null:     true,
	}, nil
}

func length(ctx context.Context, scope *Scope, pos ast.Position, args Value) (Value, error) {
	v, ok, err := args.Index(ctx, numberScalar(0, pos))
	if err != nil {
		return nil, err
	}
	l, ok := v.(Length)
	if !ok {
		t, _ := v.Type(ctx)
		return nil, fmt.Errorf("type %s does not support length", t)
	}
	i, err := l.Len(ctx)
	if err != nil {
		return nil, err
	}
	return numberScalar(i, pos), nil
}

func numberScalar(val int, pos ast.Position) Value {
	s := (ast.Number)(strconv.Itoa(val))
	return &Scalar{
		Position: pos,
		Number:   &s,
	}
}
