package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/acorn-io/aml/parser/ast"
)

type ErrInvalidLookup struct {
	Position ast.Position
	Scope    *Scope
}

type Type string

const (
	TypeString  = Type("string")
	TypeBool    = Type("bool")
	TypeArray   = Type("array")
	TypeObject  = Type("object")
	TypeNumber  = Type("number")
	TypeNull    = Type("null")
	TypeBuiltin = Type("builtin")
)

type Value interface {
	GetPosition() ast.Position
	Type(ctx context.Context) (Type, error)
	Lookup(ctx context.Context, key string) (Value, bool, error)
	Interface(ctx context.Context) (any, error)
}

type Indexable interface {
	Index(ctx context.Context, val Value) (Value, error)
}

type Sliceable interface {
	Slice(ctx context.Context, start, end Value) (Value, error)
}

type Callable interface {
	Call(ctx context.Context, scope *Scope, pos ast.Position, args []KeyValue) (Value, error)
}

type ObjectValue interface {
	Keys(ctx context.Context) ([]string, error)
	GetScope(ctx context.Context) (*Scope, error)
	GetFields(ctx context.Context) ([]*ast.Field, error)
	KeyValues(ctx context.Context) ([]KeyValue, error)
	GetPosition() ast.Position
}

type Iterator interface {
	Next() (Value, bool, error)
}

type Length interface {
	Len(ctx context.Context) (int, error)
}

type ArrayValue interface {
	Iterator(ctx context.Context) (Iterator, error)
	Empty(ctx context.Context) (bool, error)
}

type Locals struct {
	Values []KeyValue
}

func (l *Locals) Keys(ctx context.Context) (result []string, _ error) {
	for _, v := range l.Values {
		result = append(result, v.Key)
	}
	return
}

func (l *Locals) GetScope(ctx context.Context) (*Scope, error) {
	return &Scope{}, nil
}

func (l *Locals) GetFields(ctx context.Context) (result []*ast.Field, _ error) {
	for _, kv := range l.Values {
		result = append(result, &ast.Field{
			Position:    l.GetPosition(),
			StaticKey:   kv.Key,
			StaticValue: kv.Value,
		})
	}
	return
}

func (l *Locals) KeyValues(ctx context.Context) ([]KeyValue, error) {
	return l.Values, nil
}

func (l *Locals) GetPosition() ast.Position {
	return ast.Position{}
}

func (l *Locals) Add(key string, value Value) {
	if key == "" {
		return
	}
	l.Values = append(l.Values, KeyValue{
		Key:   key,
		Value: value,
	})
}

func (l *Locals) Type(ctx context.Context) (Type, error) {
	return TypeObject, fmt.Errorf("type unsupported")
}

func (l *Locals) Lookup(ctx context.Context, key string) (Value, bool, error) {
	for _, k := range l.Values {
		if k.Key == key {
			return k.Value, true, nil
		}
	}
	return nil, false, nil
}

func (l *Locals) Index(ctx context.Context, val Value) (Value, error) {
	return nil, fmt.Errorf("index unsupported")
}

func (l *Locals) Interface(ctx context.Context) (interface{}, error) {
	return nil, fmt.Errorf("interface unsupported")
}

type KeyValue struct {
	Key   string
	Value Value
}

type Scalar struct {
	Position ast.Position
	Null     bool
	Bool     *bool
	String   *string
	Number   *ast.Number
}

func (s *Scalar) GetPosition() ast.Position {
	return s.Position
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

func (s *Scalar) Lookup(ctx context.Context, key string) (_ Value, _ bool, err error) {
	return nil, false, nil
}

func ToObject(scope *Scope, v *ast.Object) *ObjectReference {
	return &ObjectReference{
		Position: v.Position,
		Scope:    scope,
		Fields:   v.Fields,
	}
}

func Eval(ctx context.Context, scope *Scope, v *ast.Value) (Value, error) {
	b, err := NewBuiltin(ctx)
	if err != nil {
		return nil, err
	}
	return ToValue(ctx, scope.Push(b), v)
}

func ToValue(ctx context.Context, scope *Scope, v *ast.Value) (Value, error) {
	ret := &Scalar{
		Position: v.Position,
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

type Map struct {
	Position ast.Position
	Scope    *Scope
	Data     map[string]interface{}
}

func (m *Map) Keys(ctx context.Context) ([]string, error) {
	var keys []string
	for k := range m.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

func (m *Map) GetScope(ctx context.Context) (*Scope, error) {
	return m.Scope, nil
}

func (m *Map) GetFields(ctx context.Context) (result []*ast.Field, _ error) {
	kvs, err := m.KeyValues(ctx)
	if err != nil {
		return nil, err
	}
	for _, kv := range kvs {
		result = append(result, &ast.Field{
			Position:    m.Position,
			StaticKey:   kv.Key,
			StaticValue: kv.Value,
		})
	}
	return
}

func anyToValue(scope *Scope, pos ast.Position, val any) (Value, error) {
	if val == nil {
		return &Scalar{
			Position: pos,
			Null:     true,
		}, nil
	}
	switch v := val.(type) {
	case []interface{}:
		var values []Value
		for _, item := range v {
			newValue, err := anyToValue(scope, pos, item)
			if err != nil {
				return nil, err
			}
			values = append(values, newValue)
		}
		return &Array{
			Position: pos,
			Values:   values,
		}, nil
	case map[string]interface{}:
		return &Map{
			Position: pos,
			Scope:    scope,
			Data:     v,
		}, nil
	case string:
		return &Scalar{
			Position: pos,
			String:   &v,
		}, nil
	case int:
		return &Scalar{
			Position: pos,
			Number:   intToNumer(int64(v)),
		}, nil
	case int64:
		return &Scalar{
			Position: pos,
			Number:   intToNumer(v),
		}, nil
	case float64:
		return &Scalar{
			Position: pos,
			Number:   floatToNumer(v),
		}, nil
	case bool:
		return &Scalar{
			Position: pos,
			Bool:     &v,
		}, nil
	default:
		return nil, fmt.Errorf("invalid type: %T", v)
	}
}

func (m *Map) KeyValues(ctx context.Context) (result []KeyValue, err error) {
	keys, err := m.Keys(ctx)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		val := m.Data[key]
		valVal, err := anyToValue(m.Scope, m.Position, val)
		if err != nil {
			return nil, err
		}

		result = append(result, KeyValue{
			Key:   key,
			Value: valVal,
		})
	}

	return
}

func floatToNumer(i float64) *ast.Number {
	s := strconv.FormatFloat(i, 'e', -1, 64)
	v := ast.Number(s)
	return &v
}

func intToNumer(i int64) *ast.Number {
	s := strconv.FormatInt(i, 10)
	v := ast.Number(s)
	return &v
}

func (m *Map) GetPosition() ast.Position {
	return m.Position
}

func (m *Map) Type(ctx context.Context) (Type, error) {
	return TypeObject, nil
}

func (m *Map) Lookup(ctx context.Context, key string) (Value, bool, error) {
	v, ok := m.Data[key]
	if !ok {
		return nil, false, nil
	}
	ret, err := anyToValue(m.Scope, m.Position, v)
	return ret, true, err
}

func (m *Map) Interface(ctx context.Context) (any, error) {
	return m.Data, nil
}
