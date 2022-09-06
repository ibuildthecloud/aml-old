package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/acorn-io/aml/parser/ast"
)

func argStringArray(ctx context.Context, v []Value, index int) ([]string, error) {
	if err := expectArgs(index+1, v); err != nil {
		return nil, err
	}
	vt, err := v[index].Type(ctx)
	if err != nil {
		return nil, err
	}
	if vt != TypeArray {
		return nil, fmt.Errorf("expected string argument at index %d, got: %s", index, vt)
	}
	var result []string
	a := v[index].(ArrayValue)
	iter, err := a.Iterator(ctx)
	if err != nil {
		return nil, err
	}
	for {
		n, cont, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !cont {
			break
		}
		vt, err := n.Type(ctx)
		if err != nil {
			return nil, err
		}
		if vt != TypeString {
			return nil, fmt.Errorf("expected array of strings but got type: %s", vt)
		}
		o, err := n.Interface(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, o.(string))
	}

	return result, nil
}

func argNumber(ctx context.Context, v []Value, index int) (int64, float64, bool, error) {
	if err := expectArgs(index+1, v); err != nil {
		return 0, 0, false, err
	}
	vt, err := v[index].Type(ctx)
	if err != nil {
		return 0, 0, false, err
	}
	if vt != TypeNumber {
		return 0, 0, false, fmt.Errorf("expected string argument at index %d, got: %s", index, vt)
	}
	s, err := v[index].Interface(ctx)
	if err != nil {
		return 0, 0, false, err
	}
	i, err := s.(json.Number).Int64()
	if err == nil {
		return i, 0, true, err
	}
	f, err := s.(json.Number).Float64()
	return 0, f, false, err
}

func argInt(ctx context.Context, v []Value, index int) (int, error) {
	if err := expectArgs(index+1, v); err != nil {
		return 0, err
	}
	vt, err := v[index].Type(ctx)
	if err != nil {
		return 0, err
	}
	if vt != TypeNumber {
		return 0, fmt.Errorf("expected string argument at index %d, got: %s", index, vt)
	}
	s, err := v[index].Interface(ctx)
	if err != nil {
		return 0, err
	}
	i, err := s.(json.Number).Int64()
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

func argString(ctx context.Context, v []Value, index int) (string, error) {
	if err := expectArgs(index+1, v); err != nil {
		return "", err
	}
	vt, err := v[index].Type(ctx)
	if err != nil {
		return "", err
	}
	if vt != TypeString {
		return "", fmt.Errorf("expected string argument at index %d, got: %s", index, vt)
	}
	s, err := v[index].Interface(ctx)
	if err != nil {
		return "", err
	}
	return s.(string), nil
}

func StringArray(pos ast.Position, s ...string) Value {
	array := &Array{
		Position: pos,
	}
	for _, s := range s {
		s := s
		array.Values = append(array.Values, &Scalar{
			Position: pos,
			String:   &s,
		})
	}
	return array
}

func StringScalar(pos ast.Position, s string) Value {
	return &Scalar{
		Position: pos,
		String:   &s,
	}
}

func Slice(ctx context.Context, v, start, end Value) (Value, error) {
	if sliceable, ok := v.(Sliceable); ok {
		return sliceable.Slice(ctx, start, end)
	}
	t, _ := v.Type(ctx)
	return nil, fmt.Errorf("type %s is not sliceable", t)
}

func IntScalar(pos ast.Position, val int) Value {
	s := (ast.Number)(strconv.Itoa(val))
	return &Scalar{
		Position: pos,
		Number:   &s,
	}
}

func FloatScalar(pos ast.Position, val float64) Value {
	s := (ast.Number)(strconv.FormatFloat(val, 'f', -1, 64))
	return &Scalar{
		Position: pos,
		Number:   &s,
	}
}

func expectBool(ctx context.Context, v Value) (bool, error) {
	vt, err := v.Type(ctx)
	if err != nil {
		return false, err
	}
	if vt != TypeBool {
		return false, fmt.Errorf("expected bool type, got: %s", vt)
	}
	b, err := v.Interface(ctx)
	if err != nil {
		return false, err
	}
	return b.(bool), nil
}
