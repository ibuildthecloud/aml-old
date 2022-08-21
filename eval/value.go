package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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

type Value interface {
	Type(ctx context.Context) (Type, error)
	Call(ctx context.Context, args ...Value) (Value, error)
	Lookup(ctx context.Context, key string) (Value, bool, error)
	Index(ctx context.Context, val Value) (Value, bool, error)
	Interface(ctx context.Context) (interface{}, error)
}

// Casting an object to this interface is not enough to determine a value
// is an Object you must first check that Value.Type() returns TypeObject
type ObjectValue interface {
	Keys(ctx context.Context) ([]string, error)
	GetScope(ctx context.Context) (*Scope, error)
	GetFields(ctx context.Context) ([]*ast.Field, error)
	GetPosition(ctx context.Context) (ast.Position, error)
	Merge(ctx context.Context, val ObjectValue) (Value, error)
}

type Iterator interface {
	Next() (Value, bool, error)
}

type ArrayValue interface {
	Iterator(ctx context.Context) (Iterator, error)
}

type Locals struct {
	values map[string]Local
	order  []string
}

func (l *Locals) Add(v Local) {
	if l.values == nil {
		l.values = map[string]Local{}
	}
	if _, ok := l.values[v.Key]; !ok {
		l.order = append(l.order, v.Key)
	}
	l.values[v.Key] = v
}

func (l *Locals) Get(key string) (Local, bool) {
	v, ok := l.values[key]
	return v, ok
}

func (l *Locals) Keys() []string {
	return l.order
}

func (l *Locals) Type(ctx context.Context) (Type, error) {
	return TypeObject, fmt.Errorf("type unsupported")
}

func (l *Locals) Lookup(ctx context.Context, key string) (Value, bool, error) {
	v, ok := l.Get(key)
	return v.Value, ok, nil
}

func (l *Locals) Index(ctx context.Context, val Value) (Value, bool, error) {
	return nil, false, fmt.Errorf("index unsupported")
}

func (l *Locals) Interface(ctx context.Context) (interface{}, error) {
	return nil, fmt.Errorf("interface unsupported")
}

func (l *Locals) Call(ctx context.Context, args ...Value) (_ Value, err error) {
	return nil, fmt.Errorf("call unsupported")
}

type Local struct {
	Key   string
	Value Value
}

type Scalar struct {
	Position ast.Position
	Scope    *Scope
	Null     bool
	Bool     *bool
	String   *string
	Number   *ast.Number
}

func (s *Scalar) Call(ctx context.Context, args ...Value) (_ Value, err error) {
	return nil, fmt.Errorf("can not call on a scalar")
}

func (s *Scalar) Type(ctx context.Context) (Type, error) {
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

func (s *Scalar) Interface(ctx context.Context) (interface{}, error) {
	if s.Null {
		return nil, nil
	} else if s.Bool != nil {
		return *s.Bool, nil
	} else if s.String != nil {
		return *s.String, nil
	} else if s.Number != nil {
		i, err := s.Number.Int64()
		if err == nil {
			return json.Number(strconv.FormatInt(i, 10)), nil
		}
		f, err := s.Number.Float64()
		if err != nil {
			return nil, err
		}
		d, err := json.Marshal(f)
		if err != nil {
			return nil, err
		}
		return json.Number(d), nil
	}
	panic(fmt.Sprintf("Invalid Scalar, no value set: %s", s.Position))
}

func (s *Scalar) Index(ctx context.Context, val Value) (_ Value, _ bool, err error) {
	return nil, false, nil
}

func (s *Scalar) Lookup(ctx context.Context, key string) (_ Value, _ bool, err error) {
	return nil, false, nil
}

func ToObject(scope *Scope, v *ast.Object) *ObjectReference {
	return &ObjectReference{
		Position: v.Position,
		Scope:    scope,
		object:   v,
	}
}

func ToValue(ctx context.Context, scope *Scope, v *ast.Value) (Value, error) {
	ret := &Scalar{
		Position: v.Position,
		Scope:    scope,
		Null:     v.Null,
		Bool:     v.Bool,
		Number:   v.Number,
	}
	switch {
	case v.String != nil:
		s, err := EvaluateString(ctx, scope, v.String)
		if err != nil {
			return nil, err
		}
		ret.String = &s
	case v.Object != nil:
		return ToObject(scope, v.Object), nil
	case v.Array != nil:
		return &Array{
			Position: v.Position,
			Scope:    scope,
			array:    v.Array,
		}, nil
	case v.Expression != nil:
		return EvaluateExpression(ctx, scope, v.Expression)
	case v.ListComprehension != nil:
		return EvaluateList(ctx, scope, v.ListComprehension)
	}
	return ret, nil
}
