package eval

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type ErrInvalidLookup struct {
	Position ast.Position
	Scope    *Scope
}

type Reference interface {
	Lookup(key string) (Reference, error)
	Resolve(ctx context.Context) (Value, error)
}

type Value interface {
	Interface() interface{}
}

type Object struct {
	data map[string]interface{}
}

func (o *Object) Interface() interface{} {
	return o.data
}

type Scalar struct {
	Position ast.Position
	Scope    *Scope
	Null     bool
	Bool     *bool
	String   *string
	Number   *json.Number
}

func (s *Scalar) Interface() interface{} {
	if s.Null {
		return nil
	} else if s.Bool != nil {
		return *s.Bool
	} else if s.String != nil {
		return *s.String
	} else if s.Number != nil {
		return *s.Number
	}
	panic(fmt.Sprintf("Invalid Scalar, no value set: %s", s.Position))
}

func (s *Scalar) Lookup(key string) (Reference, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Scalar) Resolve(ctx context.Context) (Value, error) {
	return s, nil
}

func ToValue(ctx context.Context, v ast.Value) (Value, error) {
	return toReference(&Scope{}, &v).Resolve(ctx)
}

func toReference(scope *Scope, v *ast.Value) Reference {
	ret := &Scalar{
		Position: v.Position,
		Scope:    scope,
		Null:     v.Null,
		Bool:     v.Bool,
		String:   v.String,
		Number:   v.Number,
	}
	switch {
	case v.Object != nil:
		return &ObjectReference{
			Position: v.Position,
			Scope:    scope,
			object:   v.Object,
		}
	case v.Array != nil:
	case v.Expression != nil:
	}
	return ret
}
