package eval

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/acorn-io/aml/parser"
	"github.com/acorn-io/aml/parser/ast"
	"gopkg.in/yaml.v3"
)

//go:embed std.aml
var stdFile embed.FS

var Globals = []KeyValue{
	{
		Key:   "len",
		Value: MethodFunc(length),
	},
	{
		Key: "number",
		Value: AbstractType{
			objectType: TypeNumber,
		},
	},
	{
		Key: "string",
		Value: AbstractType{
			objectType: TypeString,
		},
	},
	{
		Key: "bool",
		Value: AbstractType{
			objectType: TypeBool,
		},
	},
	{
		Key: "array",
		Value: AbstractArray{
			AbstractType: AbstractType{
				objectType: TypeArray,
			},
		},
	},
	{
		Key: "object",
		Value: AbstractArray{
			AbstractType: AbstractType{
				objectType: TypeObject,
			},
		},
	},
}

var Std = []KeyValue{
	{Key: "splitHostPort", Value: MethodFunc(splitHostPort)},
	{Key: "joinHostPort", Value: MethodFunc(joinHostPort)},
	{Key: "base64decode", Value: MethodFunc(base64decode)},
	{Key: "base64", Value: MethodFunc(base64encode)},
	{Key: "atoi", Value: MethodFunc(atoi)},
	{Key: "fileExt", Value: MethodFunc(fileExt)},
	{Key: "basename", Value: MethodFunc(basename)},
	{Key: "dirname", Value: MethodFunc(dirname)},
	{Key: "pathJoin", Value: MethodFunc(pathJoin)},
	{Key: "sha1sum", Value: MethodFunc(sha1sum)},
	{Key: "sha256sum", Value: MethodFunc(sha256sum)},
	{Key: "sha512sum", Value: MethodFunc(sha512sum)},
	{Key: "toHex", Value: MethodFunc(toHex)},
	{Key: "fromHex", Value: MethodFunc(fromHex)},
	{Key: "toJSON", Value: MethodFunc(toJSON)},
	{Key: "fromJSON", Value: MethodFunc(fromJSON)},
	{Key: "toYAML", Value: MethodFunc(toYAML)},
	{Key: "fromYAML", Value: MethodFunc(fromYAML)},
	{Key: "error", Value: MethodFunc(err)},
	{Key: "toTitle", Value: MethodFunc(toTitle)},
	{Key: "toUpper", Value: MethodFunc(toUpper)},
	{Key: "toLower", Value: MethodFunc(toLower)},
	{Key: "startsWith", Value: MethodFunc(startsWith)},
	{Key: "endsWith", Value: MethodFunc(endsWith)},
	{Key: "trim", Value: MethodFunc(trim)},
	{Key: "trimPrefix", Value: MethodFunc(trimPrefix)},
	{Key: "trimSuffix", Value: MethodFunc(trimSuffix)},
	{Key: "isString", Value: MethodFunc(isString)},
	{Key: "isNumber", Value: MethodFunc(isNumber)},
	{Key: "isBool", Value: MethodFunc(isBool)},
	{Key: "isArray", Value: MethodFunc(isArray)},
	{Key: "isObject", Value: MethodFunc(isObject)},
	{Key: "join", Value: MethodFunc(join)},
	{Key: "replace", Value: MethodFunc(replace)},
	{Key: "indexOf", Value: MethodFunc(indexOf)},
	{Key: "split", Value: MethodFunc(split)},
	{Key: "range", Value: MethodFunc(numRange)},
	{Key: "_sort", Value: MethodFunc(_sort)},
}

type Builtin struct {
	Values map[string]Value
}

func NewBuiltin(ctx context.Context) (Value, error) {
	globals := &Locals{
		Values: Globals,
	}
	globalScope := NewScope(globals)

	f, err := stdFile.Open("std.aml")
	if err != nil {
		return nil, err
	}
	v, err := parser.ParseReader("std.aml", f, parser.GlobalStore("source", "std.aml"))
	if err != nil {
		return nil, err
	}
	std, err := MergeObjects(ctx, ToObject(globalScope, v.(*ast.Value).Object), &Locals{
		Values: Std,
	})
	if err != nil {
		return nil, err
	}
	return MergeObjects(ctx, globals, &Locals{
		Values: []KeyValue{
			{
				Key:   "std",
				Value: std,
			},
		},
	})
}

func arrayToValueSlice(ctx context.Context, arr Value) (result []Value, _ error) {
	t, err := arr.Type(ctx)
	if err != nil {
		return nil, err
	}
	if t != TypeArray {
		return nil, fmt.Errorf("expected type array but got %s", t)
	}
	iter, err := arr.(ArrayValue).Iterator(ctx)
	if err != nil {
		return nil, err
	}
	for {
		val, cont, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !cont {
			break
		}
		result = append(result, val)
	}
	return result, nil
}

func _sort(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	if err := expectArgs(2, args); err != nil {
		return nil, err
	}
	vals, err := arrayToValueSlice(ctx, args[0])
	if err != nil {
		return nil, err
	}
	comp, ok := args[1].(Callable)
	if !ok {
		t, _ := args[1].Type(ctx)
		return nil, fmt.Errorf("expected callable but get type: %s", t)
	}
	var sortError []error
	sort.Slice(vals, func(i, j int) bool {
		ret, err := comp.Call(ctx, scope, pos, []KeyValue{
			{Value: vals[i]},
			{Value: vals[j]},
		})
		if err == nil {
			v, err := ret.Interface(ctx)
			if err != nil {
				sortError = append(sortError, err)
				return false
			}
			b, ok := v.(bool)
			if ok {
				return b
			}
			sortError = append(sortError, fmt.Errorf("expected bool result, got %v", v))
		} else if err != nil {
			sortError = append(sortError, err)
		}
		return false
	})
	if len(sortError) > 0 {
		return nil, sortError[0]
	}
	return &Array{
		Position: pos,
		Values:   vals,
	}, nil
}

func err(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	var val []string
	for _, v := range args {
		x, err := v.Interface(ctx)
		if err == nil {
			val = append(val, fmt.Sprint(x))
		}
	}
	return nil, fmt.Errorf(strings.Join(val, ","))
}

func isString(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	return isType(ctx, pos, TypeString, args...)
}

func isNumber(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	return isType(ctx, pos, TypeNumber, args...)
}

func isBool(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	return isType(ctx, pos, TypeBool, args...)
}

func isArray(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	return isType(ctx, pos, TypeArray, args...)
}

func isObject(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	return isType(ctx, pos, TypeObject, args...)
}

func join(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	arr, err := argStringArray(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	ret := strings.Join(arr, s)
	return &Scalar{
		Position: pos,
		String:   &ret,
	}, nil
}

func expectArgs(count int, args []Value) error {
	if len(args) < count {
		return fmt.Errorf("expected at least %d arguments, got %d", count, len(args))
	}
	return nil
}

func indexOf(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	if err := expectArgs(2, args); err != nil {
		return nil, err
	}
	firstType, err := args[0].Type(ctx)
	if err != nil {
		return nil, err
	}
	secondType, err := args[1].Type(ctx)
	if err != nil {
		return nil, err
	}
	if firstType == TypeString && secondType == TypeString {
		s, err := argString(ctx, args, 0)
		if err != nil {
			return nil, err
		}
		s2, err := argString(ctx, args, 1)
		if err != nil {
			return nil, err
		}
		ret := strings.Index(s, s2)
		return IntScalar(pos, ret), nil
	}
	if firstType == TypeArray {
		checkValue, err := args[1].Interface(ctx)
		if err != nil {
			return nil, err
		}
		iter, err := args[0].(ArrayValue).Iterator(ctx)
		if err != nil {
			return nil, err
		}
		for i := 0; ; i++ {
			v, cont, err := iter.Next()
			if err != nil {
				return nil, err
			}
			if !cont {
				break
			}
			val, err := v.Interface(ctx)
			if err != nil {
				return nil, err
			}
			if val == checkValue {
				return IntScalar(pos, i), nil
			}
		}
		return IntScalar(pos, -1), nil
	}
	return nil, fmt.Errorf("invalid argument types [%s, %s] expected [string, string] or [array, any])", firstType, secondType)
}

func numRange(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	start, startf, startInt := int64(0), 0.0, true
	step, stepf, stepInt := int64(1), 1.0, true
	max, maxf, maxInt, err := argNumber(ctx, args, 0)
	if err != nil {
		return nil, err
	}

	if len(args) > 1 {
		start = max
		startf = maxf
		startInt = maxInt
		max, maxf, maxInt, err = argNumber(ctx, args, 1)
		if err != nil {
			return nil, err
		}
	}

	if len(args) > 2 {
		step, stepf, stepInt, err = argNumber(ctx, args, 2)
		if err != nil {
			return nil, err
		}
	}

	if (stepInt && step == 0) || (!stepInt && stepf == 0.0) {
		return nil, fmt.Errorf("invalid step value 0")
	}

	var values []Value
	if stepInt && startInt && maxInt {
		if step > 0 {
			for i := start; i < max; i += step {
				tick(ctx)
				values = append(values, IntScalar(pos, int(i)))
			}
		} else {
			for i := start; i > max; i += step {
				tick(ctx)
				values = append(values, IntScalar(pos, int(i)))
			}
		}
	} else {
		if stepInt {
			stepf = float64(step)
		}
		if startInt {
			startf = float64(start)
		}
		if maxInt {
			maxf = float64(max)
		}
		if stepf > 0 {
			for i := startf; i < maxf; i += stepf {
				tick(ctx)
				values = append(values, FloatScalar(pos, i))
			}
		} else {
			for i := startf; i > maxf; i += stepf {
				tick(ctx)
				values = append(values, FloatScalar(pos, i))
			}
		}
	}

	return &Array{
		Position: pos,
		Values:   values,
	}, nil
}

func split(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	tok, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	count := -1
	if len(args) > 2 {
		c, err := argInt(ctx, args, 2)
		if err != nil {
			return nil, err
		}
		count = c
	}
	ret := strings.SplitN(s, tok, count)
	return StringArray(pos, ret...), nil
}

func replace(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	find, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	replace, err := argString(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	count := -1
	if len(args) > 3 {
		n, err := argInt(ctx, args, 3)
		if err != nil {
			return nil, err
		}
		count = n
	}
	ret := strings.Replace(s, find, replace, count)
	return &Scalar{
		Position: pos,
		String:   &ret,
	}, nil
}

func isType(ctx context.Context, pos ast.Position, objectType Type, args ...Value) (Value, error) {
	if err := expectArgs(1, args); err != nil {
		return nil, err
	}
	t, err := args[0].Type(ctx)
	if err != nil {
		return nil, err
	}
	ret := t == objectType
	return &Scalar{
		Position: pos,
		Bool:     &ret,
	}, nil

}

func startsWith(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	prefix, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	result := strings.HasPrefix(s, prefix)
	return &Scalar{
		Position: pos,
		Bool:     &result,
	}, nil
}

func endsWith(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	suffix, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	result := strings.HasSuffix(s, suffix)
	return &Scalar{
		Position: pos,
		Bool:     &result,
	}, nil
}

func trim(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s = strings.TrimSpace(s)
	return &Scalar{
		Position: pos,
		String:   &s,
	}, nil
}

func trimSuffix(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	suffix, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	s = strings.TrimSuffix(s, suffix)
	return &Scalar{
		Position: pos,
		String:   &s,
	}, nil
}

func trimPrefix(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	prefix, err := argString(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	s = strings.TrimPrefix(s, prefix)
	return &Scalar{
		Position: pos,
		String:   &s,
	}, nil
}

func toTitle(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	prev := ' '
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(prev) {
			prev = r
			return unicode.ToTitle(r)
		}
		prev = r
		return r
	}, s)

	return &Scalar{
		Position: pos,
		String:   &s,
	}, nil
}

func toUpper(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s = strings.ToUpper(s)
	return &Scalar{
		Position: pos,
		String:   &s,
	}, nil
}

func toLower(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s = strings.ToLower(s)
	return &Scalar{
		Position: pos,
		String:   &s,
	}, nil
}

func fromYAML(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(s), &data)
	if err != nil {
		return nil, err
	}
	return &Map{
		Position: pos,
		Scope:    scope,
		Data:     data,
	}, nil
}

func toYAML(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	if err := expectArgs(1, args); err != nil {
		return nil, err
	}
	v, err := args[0].Interface(ctx)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	return StringScalar(pos, string(data)), nil
}

func fromJSON(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{}
	err = json.Unmarshal([]byte(s), &data)
	if err != nil {
		return nil, err
	}
	return &Map{
		Position: pos,
		Scope:    scope,
		Data:     data,
	}, nil
}

func toJSON(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	if err := expectArgs(1, args); err != nil {
		return nil, err
	}
	v, err := args[0].Interface(ctx)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return StringScalar(pos, string(data)), nil
}

func fromHex(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return StringScalar(pos, string(b)), nil
}

func toHex(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	return StringScalar(pos, hex.EncodeToString([]byte(s))), nil
}

func sha512sum(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	d := sha512.Sum512([]byte(s))
	return StringScalar(pos, hex.EncodeToString(d[:])), nil
}

func sha256sum(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	d := sha256.Sum256([]byte(s))
	return StringScalar(pos, hex.EncodeToString(d[:])), nil
}

func sha1sum(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	d := sha1.Sum([]byte(s))
	return StringScalar(pos, hex.EncodeToString(d[:])), nil
}

func pathJoin(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	arr, err := argStringArray(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	sep := string([]byte{filepath.Separator})
	newSep := "/"
	ret := filepath.Join(arr...)
	if len(args) > 1 {
		newSep, err = argString(ctx, args, 1)
		if err != nil {
			return nil, err
		}
	}
	if sep != newSep {
		ret = strings.ReplaceAll(ret, sep, newSep)
	}
	return &Scalar{
		Position: pos,
		String:   &ret,
	}, nil
}

func dirname(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s = filepath.Dir(s)
	return StringScalar(pos, s), nil
}

func basename(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s = filepath.Base(s)
	return StringScalar(pos, s), nil
}

func fileExt(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s = filepath.Ext(s)
	return StringScalar(pos, s), nil
}

func atoi(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return IntScalar(pos, i), nil
}

func length(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	if err := expectArgs(1, args); err != nil {
		return nil, err
	}
	l, ok := args[0].(Length)
	if !ok {
		t, _ := args[0].Type(ctx)
		return nil, fmt.Errorf("type %s does not support length", t)
	}
	i, err := l.Len(ctx)
	if err != nil {
		return nil, err
	}
	return IntScalar(pos, i), nil
}

func base64encode(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	vStr, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s := base64.StdEncoding.EncodeToString([]byte(vStr))
	return StringScalar(pos, s), nil
}

func base64decode(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	vStr, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	s, err := base64.StdEncoding.DecodeString(vStr)
	if err != nil {
		return nil, err
	}
	return StringScalar(pos, string(s)), nil
}

func joinHostPort(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	if err := expectArgs(2, args); err != nil {
		return nil, err
	}
	vStr, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	vStr2, err := argString(ctx, args, 1)
	if err != nil {
		n, numErr := argInt(ctx, args, 1)
		if numErr == nil {
			vStr2 = strconv.Itoa(n)
		} else {
			return nil, fmt.Errorf("[%s] or [%s]", err, numErr)
		}
	}
	result := net.JoinHostPort(vStr, vStr2)
	return StringScalar(pos, result), nil

}

func splitHostPort(ctx context.Context, scope *Scope, pos ast.Position, args ...Value) (Value, error) {
	s, err := argString(ctx, args, 0)
	if err != nil {
		return nil, err
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return nil, err
	}
	return StringArray(pos, host, port), nil
}
