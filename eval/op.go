package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/acorn-io/aml/parser/ast"
)

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

func BinaryOp(ctx context.Context, _ *Scope, pos ast.Position, op string, left, right Value) (_ Value, err error) {
	defer func() {
		err = wrapErr(pos, err)
	}()

	lt, err := left.Type(ctx)
	if err != nil {
		return nil, err
	}

	rt, err := right.Type(ctx)
	if err != nil {
		return nil, err
	}

	if op == "==" && (lt == TypeNull || rt == TypeNull) {
		b := lt == rt
		return &Scalar{
			Position: pos,
			Bool:     &b,
		}, nil
	}

	if op == "!=" && (lt == TypeNull || rt == TypeNull) {
		b := lt != rt
		return &Scalar{
			Position: pos,
			Bool:     &b,
		}, nil
	}

	if lt != rt {
		return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", op, lt, rt)
	}

	if op == "+" && lt == TypeArray {
		var val []Value
		leftIter, leftErr := left.(ArrayValue).Iterator(ctx)
		if leftErr != nil {
			return nil, leftErr
		}
		rightIter, rightErr := right.(ArrayValue).Iterator(ctx)
		if rightErr != nil {
			return nil, rightErr
		}

		for {
			v, cont, err := leftIter.Next()
			if err != nil {
				return nil, err
			}
			if !cont {
				break
			}
			val = append(val, v)
		}
		for {
			v, cont, err := rightIter.Next()
			if err != nil {
				return nil, err
			}
			if !cont {
				break
			}
			val = append(val, v)
		}
		return &Array{
			Position: pos,
			Values:   val,
		}, nil
	}

	lv, err := left.Interface(ctx)
	if err != nil {
		return nil, err
	}
	rv, err := right.Interface(ctx)
	if err != nil {
		return nil, err
	}

	if op == "+" {
		if lt == TypeString {
			s := lv.(string) + rv.(string)
			return &Scalar{
				Position: pos,
				String:   &s,
			}, nil
		}
	} else if op == "&&" {
		if lt != TypeBool {
			return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", op, lt, rt)
		}
		ret := lv.(bool) && rv.(bool)
		return &Scalar{
			Position: pos,
			Bool:     &ret,
		}, nil
	} else if op == "||" {
		if lt != TypeBool {
			return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", op, lt, rt)
		}
		ret := lv.(bool) || rv.(bool)
		return &Scalar{
			Position: pos,
			Bool:     &ret,
		}, nil
	} else if op == "==" && lt != TypeNumber && rt != TypeNumber {
		ret := lv == rv
		return &Scalar{
			Position: pos,
			Bool:     &ret,
		}, nil
	} else if op == "!=" {
		ret := lv != rv
		return &Scalar{
			Position: pos,
			Bool:     &ret,
		}, nil
	} else if op == "=~" {
		regexp, err := regexp.Compile(rv.(string))
		if err != nil {
			return nil, err
		}
		ret := regexp.MatchString(lv.(string))
		return &Scalar{
			Position: pos,
			Bool:     &ret,
		}, nil
	} else if op == "!~" {
		regexp, err := regexp.Compile(rv.(string))
		if err != nil {
			return nil, err
		}
		ret := !regexp.MatchString(lv.(string))
		return &Scalar{
			Position: pos,
			Bool:     &ret,
		}, nil
	}

	if lt != TypeNumber {
		return nil, fmt.Errorf("operator %s in not compatible with types %s and %s", op, lt, rt)
	}

	lv = toNum(lv.(json.Number))
	rv = toNum(rv.(json.Number))

	flv, flvok := lv.(float64)
	frv, frvok := rv.(float64)
	ilv, _ := lv.(int64)
	irv, _ := rv.(int64)

	if flvok || frvok {
		var (
			ret  float64
			retb bool
			b    bool
		)
		if !flvok {
			flv = float64(ilv)
		} else if !frvok {
			frv = float64(irv)
		}
		switch op {
		case "*":
			ret = flv * frv
		case "/":
			ret = flv / frv
		case "+":
			ret = flv + frv
		case "-":
			ret = flv - frv
		case "<":
			retb = flv < frv
			b = true
		case "<=":
			retb = flv <= frv
			b = true
		case ">":
			retb = flv > frv
			b = true
		case ">=":
			retb = flv >= frv
			b = true
		case "==":
			retb = flv == frv
			b = true
		}
		if b {
			return &Scalar{
				Position: pos,
				Bool:     &retb,
			}, nil
		}
		s := fmt.Sprint(ret)
		return &Scalar{
			Position: pos,
			Number:   (*ast.Number)(&s),
		}, nil
	}

	var (
		ret  int64
		retb bool
		b    bool
	)
	switch op {
	case "*":
		ret = ilv * irv
	case "/":
		ret = ilv / irv
	case "+":
		ret = ilv + irv
	case "-":
		ret = ilv - irv
	case "<":
		retb = ilv < irv
		b = true
	case "<=":
		retb = ilv <= irv
		b = true
	case ">":
		retb = ilv > irv
		b = true
	case ">=":
		retb = ilv >= irv
		b = true
	case "==":
		retb = ilv == irv
		b = true
	}
	if b {
		return &Scalar{
			Position: pos,
			Bool:     &retb,
		}, nil
	}
	s := fmt.Sprint(ret)
	return &Scalar{
		Position: pos,
		Number:   (*ast.Number)(&s),
	}, nil
}

func Not(ctx context.Context, pos ast.Position, val Value) (_ Value, err error) {
	defer func() {
		err = wrapErr(pos, err)
	}()

	t, err := val.Type(ctx)
	if err != nil {
		return nil, err
	}

	scalar, ok := val.(*Scalar)
	if !ok {
		return nil, fmt.Errorf("operator ! not applicable for type: %s", t)
	} else if scalar.Bool == nil {
		return nil, wrapErr(scalar.Position, fmt.Errorf("operator ! not applicable for type: %s", t))
	}

	not := !(*scalar.Bool)
	return &Scalar{
		Position: scalar.Position,
		Bool:     &not,
	}, nil
}
