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

type Type string

const (
	TypeString = Type("string")
	TypeBool   = Type("bool")
	TypeArray  = Type("array")
	TypeObject = Type("object")
	TypeNumber = Type("number")
	TypeNull   = Type("null")
)

type Reference interface {
	Type() (Type, error)
	Lookup(key string) (Reference, error)
	Resolve(ctx context.Context) (Value, error)
}

type Value interface {
	Interface() interface{}
}

type Object struct {
	Position ast.Position
	data     map[string]interface{}
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

func (s *Scalar) Type() (Type, error) {
	if s.Null {
		return TypeNull, nil
	} else if s.Bool != nil {
		return TypeBool, nil
	} else if s.String != nil {
		return TypeString, nil
	} else if s.Number != nil {
		return TypeNumber, nil
	}

	panic(fmt.Sprintf("Invalid Scalar, no value set: %s", s.Position))
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

func (s *Scalar) Lookup(key string) (_ Reference, err error) {
	defer func() {
		err = wrapErr(s.Position, err)
	}()
	return nil, fmt.Errorf("%s can not lookup key on scalar %v", s.Position, s.Interface())
}

func (s *Scalar) Resolve(ctx context.Context) (Value, error) {
	return s, nil
}

func ToValue(ctx context.Context, v *ast.Value) (Value, error) {
	return toReference(&Scope{
		Reference: nil,
	}, v).Resolve(ctx)
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
		return &ArrayReference{
			Position: v.Position,
			Scope:    scope,
			array:    v.Array,
		}
	case v.Expression != nil:
		return &Expression{
			Position: v.Position,
			Scope:    scope,
			expr:     v.Expression,
		}
	}
	return ret
}
