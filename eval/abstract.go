package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type AbstractType struct {
	objectType Type
}

func (a AbstractType) GetPosition() ast.Position {
	return ast.Position{}
}

func (a AbstractType) Type(ctx context.Context) (Type, error) {
	return a.objectType, nil
}

func (a AbstractType) Lookup(ctx context.Context, key string) (Value, bool, error) {
	return nil, false, nil
}

func (a AbstractType) Interface(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("abstract value '%s' can not be evaluated to a value", a.objectType)
}

type AbstractArray struct {
	AbstractType
}

func (a AbstractArray) Iterator(ctx context.Context) (Iterator, error) {
	return &iter{
		index:  0,
		values: nil,
		ctx:    ctx,
	}, nil
}

func (a AbstractArray) Empty(ctx context.Context) (bool, error) {
	return true, nil
}

type AbstractObject struct {
	AbstractType
}

func (a AbstractObject) Keys(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a AbstractObject) GetScope(ctx context.Context) (*Scope, error) {
	return &Scope{}, nil
}

func (a AbstractObject) GetFields(ctx context.Context) ([]*ast.Field, error) {
	return nil, nil
}

func (a AbstractObject) KeyValues(ctx context.Context) ([]KeyValue, error) {
	return nil, nil
}
