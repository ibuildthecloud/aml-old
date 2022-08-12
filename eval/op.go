package eval

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type BinaryOp struct {
	Position ast.Position
	Op       string
	Left     Reference
	Right    Reference
}

func (b *BinaryOp) Type() (Type, error) {
	return b.Left.Type()
}

func (b *BinaryOp) Lookup(key string) (Reference, error) {
	t, _ := b.Type()
	return nil, wrapErr(b.Position, fmt.Errorf("can not lookup key %s on type %s", key, t))
}

func toNum(num json.Number) interface{} {
	i, err := num.Int64()
	if err != nil {
		f, err := num.Float64()
		if err != nil {
			panic(err.Error())
		}
		return f
	}
	return i
}

func (b *BinaryOp) Resolve(ctx context.Context) (_ Value, err error) {
	defer func() {
		err = wrapErr(b.Position, err)
	}()

	lt, err := b.Left.Type()
	if err != nil {
		return nil, err
	}

	rt, err := b.Right.Type()
	if err != nil {
		return nil, err
	}

	if lt != rt {
		return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", b.Op, lt, rt)
	}

	l, err := b.Left.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	r, err := b.Right.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	lv := l.Interface()
	rv := r.Interface()

	if b.Op == "+" {
		if lt == TypeString {
			s := lv.(string) + rv.(string)
			return &Scalar{
				Position: b.Position,
				String:   &s,
			}, nil
		} else if lt == TypeArray {
			return &ArrayValue{
				Position: b.Position,
				Value:    append(lv.([]interface{}), rv.([]interface{})...),
			}, nil
		}
	} else if b.Op == "&&" {
		if lt != TypeBool {
			return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", b.Op, lt, rt)
		}
		ret := lv.(bool) && rv.(bool)
		return &Scalar{
			Position: b.Position,
			Bool:     &ret,
		}, nil
	} else if b.Op == "||" {
		if lt != TypeBool {
			return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", b.Op, lt, rt)
		}
		ret := lv.(bool) || rv.(bool)
		return &Scalar{
			Position: b.Position,
			Bool:     &ret,
		}, nil
	}

	if lt != TypeNumber {
		return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", b.Op, lt, rt)
	}

	lv = toNum(lv.(json.Number))
	rv = toNum(rv.(json.Number))

	flv, flvok := lv.(float64)
	frv, frvok := rv.(float64)
	ilv, _ := lv.(int64)
	irv, _ := rv.(int64)

	if flvok || frvok {
		var ret float64
		if !flvok {
			flv = float64(ilv)
		} else if !frvok {
			frv = float64(irv)
		}
		switch b.Op {
		case "*":
			ret = flv * frv
		case "/":
			ret = flv / frv
		case "+":
			ret = flv + frv
		case "-":
			ret = flv - frv
		}
		s := fmt.Sprint(ret)
		return &Scalar{
			Position: b.Position,
			Number:   (*json.Number)(&s),
		}, nil
	}

	var ret int64
	switch b.Op {
	case "*":
		ret = ilv * irv
	case "/":
		ret = ilv / irv
	case "+":
		ret = ilv + irv
	case "-":
		ret = ilv - irv
	}
	s := fmt.Sprint(ret)
	return &Scalar{
		Position: b.Position,
		Number:   (*json.Number)(&s),
	}, nil
}

type Not struct {
	Position ast.Position
	ref      Reference
}

func (n Not) Type() (Type, error) {
	return TypeBool, nil
}

func (n Not) Lookup(key string) (_ Reference, err error) {
	defer func() {
		err = wrapErr(n.Position, err)
	}()
	return n.ref.Lookup(key)
}

func (n Not) Resolve(ctx context.Context) (_ Value, err error) {
	defer func() {
		err = wrapErr(n.Position, err)
	}()

	t, err := n.ref.Type()
	if err != nil {
		return nil, err
	}

	v, err := n.ref.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	val, ok := v.(*Scalar)
	if !ok {
		return nil, fmt.Errorf("operator ! not applicable for type: %s", t)
	} else if val.Bool == nil {
		return nil, wrapErr(val.Position, fmt.Errorf("operator ! not applicable for type: %s", t))
	}

	not := !(*val.Bool)
	return &Scalar{
		Position: val.Position,
		Bool:     &not,
	}, nil
}
