package eval

import (
	"context"
	"fmt"
	"regexp"

	"github.com/acorn-io/aml/parser/ast"
)

const EmbeddedKey = " __embedded__ "

type FieldReference struct {
	Scope     *Scope
	Field     *ast.Field
	resolving bool

	values    map[string]Value
	misses    map[string]bool
	condition *bool
	body      Value
	key       *string
	keys      []string
	embedded  *bool
}

func (f *FieldReference) lock() (err error) {
	defer func() {
		err = wrapErr(f.Field.Position, err)
	}()
	if f.resolving {
		return fmt.Errorf("cycle detected")
	}
	f.resolving = true
	return nil
}

func (f *FieldReference) unlock() {
	f.resolving = false
}

func (f *FieldReference) checkCondition(ctx context.Context) (_ bool, err error) {
	defer func() {
		err = wrapErr(f.Field.Position, err)
	}()

	if f.Field.If == nil || f.Field.If.Condition == nil {
		return true, nil
	}

	if f.condition != nil {
		return *f.condition, nil
	}

	if err := f.lock(); err != nil {
		return false, err
	}
	defer f.unlock()

	v, err := EvaluateExpression(ctx, f.Scope, f.Field.If.Condition)
	if err != nil {
		return false, err
	}
	vt, err := v.Type(ctx)
	if err != nil {
		return false, err
	}
	if vt != TypeBool {
		return false, fmt.Errorf("expecting boolean, expression evaluated to %v", vt)
	}
	obj, err := v.Interface(ctx)
	if err != nil {
		return false, err
	}
	b := obj.(bool)
	f.condition = &b
	return b, nil
}

func (f *FieldReference) resolveKey(ctx context.Context) (string, error) {
	if f.key != nil {
		return *f.key, nil
	}

	if err := f.lock(); err != nil {
		return "", err
	}
	defer f.unlock()

	s, err := EvaluateString(ctx, f.Scope, f.Field.Key.Name)
	if err != nil {
		return "", err
	}

	return s, nil
}

func (f *FieldReference) processKeyField(ctx context.Context, key string) (Value, bool, error) {
	if f.Field.Embedded && key == EmbeddedKey {
		v, err := ToValue(ctx, f.Scope, f.Field.Value)
		return v, true, err
	}

	if ok, err := f.matchKey(ctx, key); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	if err := f.lock(); err != nil {
		return nil, false, err
	}
	defer f.unlock()

	v, err := ToValue(ctx, f.Scope, f.Field.Value)
	return v, true, err
}

func (f *FieldReference) matchKey(ctx context.Context, key string) (bool, error) {
	if !f.Field.Key.Match {
		if ret := QuickMatch(f.Field.Key.Name, key); ret == True {
			return true, nil
		} else if ret == False {
			return false, nil
		}
	}

	s, err := f.resolveKey(ctx)
	if err != nil {
		return false, err
	}

	if f.Field.Key.Match {
		regexp, err := regexp.Compile(s)
		if err != nil {
			return false, err
		}
		return regexp.MatchString(key), nil
	}

	return s == key, nil
}

func (f *FieldReference) forBody(ctx context.Context) (Value, error) {
	if err := f.lock(); err != nil {
		return nil, err
	}
	defer f.unlock()

	list, err := EvaluateList(ctx, f.Scope, f.Field.For)
	if err != nil {
		return nil, err
	}

	iter, err := list.Iterator(ctx)
	if err != nil {
		return nil, err
	}

	// Empty node
	var result Value = EmptyObjectReference(f.Field.For.Position, f.Scope)
	for {
		n, cont, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !cont {
			break
		}
		result, err = Merge(ctx, f.Field.Position, result, n)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (f *FieldReference) getBody(ctx context.Context) (Value, error) {
	if f.body != nil {
		return f.body, nil
	}
	if f.Field.If != nil {
		f.body = ToObject(f.Scope, f.Field.If.Object)
	} else if f.Field.For != nil {
		body, err := f.forBody(ctx)
		if err != nil {
			return nil, err
		}
		f.body = body
	}
	return f.body, nil
}

func (f *FieldReference) processIfFor(ctx context.Context, key string) (Value, bool, error) {
	if ok, err := f.checkCondition(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	body, err := f.getBody(ctx)
	if err != nil {
		return nil, false, err
	}
	if key == EmbeddedKey {
		return body, true, nil
	}
	return body.Lookup(ctx, key)
}

func isEmbeddedFields(fields []*ast.Field) (ok bool, err error) {
	if len(fields) == 0 {
		return false, nil
	}
	ret := fields[0].Embedded
	for i := 1; i < len(fields); i++ {
		if fields[i].Embedded != ret {
			return false, fmt.Errorf("can not mix embedded and non-embedded object fields")
		}
	}
	return ret, nil
}

func isEmbedded(field *ast.Field) (ok bool, err error) {
	if field.If != nil {
		return isEmbeddedFields(field.If.Object.Fields)
	} else if field.For != nil {
		return isEmbeddedFields(field.For.Object.Fields)
	}
	return field.Embedded, nil
}

func (f *FieldReference) Embedded() (ok bool, err error) {
	if f.embedded != nil {
		return *f.embedded, nil
	}
	b, err := isEmbedded(f.Field)
	if err != nil {
		return false, err
	}
	f.embedded = &b
	return b, nil
}

func (f *FieldReference) Value(ctx context.Context, key string) (v Value, ok bool, err error) {
	if f.misses[key] {
		return nil, false, nil
	}

	if v, ok := f.values[key]; ok {
		return v, true, nil
	}

	if f.Field.If != nil || f.Field.For != nil {
		v, ok, err = f.processIfFor(ctx, key)
	} else {
		v, ok, err = f.processKeyField(ctx, key)
	}
	if err != nil {
		return nil, false, err
	} else if !ok {
		if f.misses == nil {
			f.misses = map[string]bool{}
		}
		f.misses[key] = true
		return nil, false, nil
	}
	if f.values == nil {
		f.values = map[string]Value{}
	}
	f.values[key] = v
	return v, true, nil
}

func (f *FieldReference) Keys(ctx context.Context) ([]string, error) {
	if f.keys != nil || f.Field.Key.Match {
		return f.keys, nil
	}

	if ok, err := f.Embedded(); err != nil {
		return nil, err
	} else if ok {
		f.keys = []string{}
		return nil, nil
	}

	if f.Field.Key.Name == nil {
		b, err := f.getBody(ctx)
		if err != nil {
			return nil, err
		}
		t, err := b.Type(ctx)
		if err != nil {
			return nil, err
		}
		if t == TypeObject {
			return b.(ObjectValue).Keys(ctx)
		}
		return nil, nil
	}

	s, err := f.resolveKey(ctx)
	return []string{s}, err
}
