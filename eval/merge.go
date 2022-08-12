package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type Merge struct {
	Position ast.Position
	Left     Reference
	Right    Reference
}

func (m *Merge) Type() (Type, error) {
	return m.Left.Type()
}

func (m *Merge) Lookup(key string) (Reference, error) {
	lv, lerr := m.Left.Lookup(key)
	rv, rerr := m.Right.Lookup(key)

	if e := (*ErrKeyNotFound)(nil); errors.As(lerr, &e) {
		if rerr == nil {
			return rv, nil
		}
		return nil, lerr
	} else if lerr != nil {
		return nil, lerr
	}

	if e := (*ErrKeyNotFound)(nil); errors.As(rerr, &e) {
		return lv, nil
	} else if rerr != nil {
		return nil, rerr
	}

	lvt, err := lv.Type()
	if err != nil {
		return nil, err
	}

	rvt, err := rv.Type()
	if err != nil {
		return nil, err
	}

	if lvt != rvt {
		return nil, wrapErr(m.Position, fmt.Errorf("can not merge incompatible types %s and %s", lvt, rvt))
	}

	if lvt == TypeObject {
		return &Merge{
			Position: m.Position,
			Left:     lv,
			Right:    rv,
		}, nil
	}

	return rv, nil
}

func (m *Merge) Resolve(ctx context.Context) (Value, error) {
	lv, lerr := m.Left.Resolve(ctx)
	rv, rerr := m.Right.Resolve(ctx)

	if e := (*ErrKeyNotFound)(nil); errors.As(lerr, &e) {
		if rerr == nil {
			return rv, nil
		}
		return nil, lerr
	} else if lerr != nil {
		return nil, lerr
	}

	if e := (*ErrKeyNotFound)(nil); errors.As(rerr, &e) {
		return lv, nil
	} else if rerr != nil {
		return nil, rerr
	}

	lvt, err := m.Left.Type()
	if err != nil {
		return nil, err
	}

	rvt, err := m.Right.Type()
	if err != nil {
		return nil, err
	}

	if lvt != rvt {
		return nil, wrapErr(m.Position, fmt.Errorf("can not merge incompatible types %s and %s", lvt, rvt))
	}

	if lvt != TypeObject {
		return rv, nil
	}

	data, err := mergeObject(lv.Interface().(map[string]interface{}), rv.Interface().(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	return &Object{
		Position: m.Position,
		data:     data,
	}, nil
}

func isCompatible(left, right interface{}) bool {
	return (isString(left) == isString(right)) &&
		(isObject(left) == isObject(right)) &&
		(isArray(left) == isArray(right)) &&
		(isNumber(left) == isNumber(right)) &&
		(isBool(left) == isBool(right))
}

func mergeObject(left, right map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	for k, v := range left {
		rv, ok := right[k]
		if !ok {
			result[k] = v
			continue
		}
		if !isCompatible(v, rv) {
			return nil, fmt.Errorf("values %v and %v are not of compatible types to merge", v, rv)
		}
		if _, ok := v.(map[string]interface{}); ok {
			if right == nil {
				result[k] = right
			} else {
				newMap, err := mergeObject(v.(map[string]interface{}), rv.(map[string]interface{}))
				if err != nil {
					return nil, err
				}
				result[k] = newMap
			}
		} else {
			result[k] = rv
		}
	}

	for k, v := range right {
		_, ok := left[k]
		if !ok {
			result[k] = v
		}
	}

	return result, nil
}

func isArray(i interface{}) bool {
	_, ok := i.([]interface{})
	return ok
}

func isBool(i interface{}) bool {
	_, ok := i.(bool)
	return ok
}

func isString(i interface{}) bool {
	_, ok := i.(string)
	return ok
}

func isNumber(i interface{}) bool {
	_, ok := i.(json.Number)
	return ok
}

func isObject(i interface{}) bool {
	_, ok := i.(map[string]interface{})
	return ok
}
