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

	values        map[string]Value
	misses        map[string]bool
	noMatch       map[string]bool
	condition     *bool
	body          Value
	key           *string
	keys          []string
	embeddedValue Value
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

func (f *FieldReference) checkCondition(ctx context.Context, key string) (_ bool, err error) {
	defer func() {
		err = wrapErr(f.Field.Position, err)
	}()

	if f.Field.If == nil || f.Field.If.Condition == nil {
		return true, nil
	}

	if f.condition != nil {
		return *f.condition, nil
	}

	if f.resolving {
		f.setNoMatch(key)
		return false, nil
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

func (f *FieldReference) resolveKey(ctx context.Context, key string) (string, bool, error) {
	if f.key != nil {
		return *f.key, true, nil
	}

	// an empty key means we don't know what we are currently looking for
	if f.resolving && key != "" {
		f.setNoMatch(key)
		return "", false, nil
	}

	if err := f.lock(); err != nil {
		return "", false, err
	}
	defer f.unlock()

	s, err := EvaluateString(ctx, f.Scope, f.Field.Key.Name)
	if err != nil {
		return "", false, err
	}

	if f.noMatch[s] {
		return "", false, fmt.Errorf("cycle detected for key evaluated to %s", s)
	}
	f.noMatch = nil

	return s, true, nil
}

func (f *FieldReference) setNoMatch(key string) {
	if f.noMatch == nil {
		f.noMatch = map[string]bool{}
	}
	f.noMatch[key] = true
}

func (f *FieldReference) lookupKeyInValue(ctx context.Context, key string, v Value) (Value, bool, error) {
	defer func() {
		f.noMatch = nil
	}()

	if key == EmbeddedKey {
		if f.noMatch[EmbeddedKey] {
			return nil, false, fmt.Errorf("cycle detected resolving embedded object")
		}
		return v, true, nil
	}

	for noMatch := range f.noMatch {
		_, ok, err := v.Lookup(ctx, key)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return nil, false, fmt.Errorf("cycle detected resolving key: %s", noMatch)
		}
	}

	return v.Lookup(ctx, key)
}

func (f *FieldReference) lookupEmbeddedKey(ctx context.Context, key string) (Value, bool, error) {
	if f.resolving {
		f.setNoMatch(key)
		return nil, false, nil
	}
	defer func() {
		f.noMatch = nil
	}()

	v, err := f.getEmbeddedValue(ctx)
	if err != nil {
		return nil, false, err
	}

	return f.lookupKeyInValue(ctx, key, v)
}

func (f *FieldReference) getEmbeddedValue(ctx context.Context) (Value, error) {
	if f.embeddedValue != nil {
		return f.embeddedValue, nil
	}

	if err := f.lock(); err != nil {
		return nil, err
	}
	defer f.unlock()

	v, err := ToValue(ctx, f.Scope, f.Field.Value)
	if err != nil {
		return nil, err
	}

	f.embeddedValue = v
	return v, nil
}

func (f *FieldReference) processKeyField(ctx context.Context, key string) (Value, bool, error) {
	if f.Field.Embedded {
		return f.lookupEmbeddedKey(ctx, key)
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
	if f.Field.Key.Name == nil {
		return false, nil
	}

	if !f.Field.Key.Match {
		if ret := QuickMatch(f.Field.Key.Name, key); ret == True {
			return true, nil
		} else if ret == False {
			return false, nil
		}
	}

	s, ok, err := f.resolveKey(ctx, key)
	if err != nil || !ok {
		return ok, err
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
	if f.resolving {
		return ToObject(f.Scope.Disallow(f.Field.For.IndexVar, f.Field.For.ValueVar), f.Field.For.Object), nil
	}

	if err := f.lock(); err != nil {
		return nil, err
	}
	defer f.unlock()

	if f.body != nil {
		return f.body, nil
	}

	list, err := EvaluateList(ctx, f.Scope, f.Field.For)
	if err != nil {
		return nil, err
	}

	iter, err := list.Iterator(ctx)
	if err != nil {
		return nil, err
	}

	result := &ObjectReference{
		Position: f.Field.For.Object.Position,
		Scope:    f.Scope,
	}

	for {
		n, cont, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !cont {
			break
		}
		result.fields = append(result.fields, &FieldReference{
			Scope: f.Scope.Push(result),
			Field: &ast.Field{
				Position: f.Field.For.Object.Position,
				Embedded: true,
			},
			embeddedValue: n,
		})
	}

	f.body = result
	return result, nil
}

func (f *FieldReference) getBody(ctx context.Context, key string) (Value, error) {
	if f.Field.If != nil {
		if f.body != nil {
			return f.body, nil
		}
		f.body = ToObject(f.Scope, f.Field.If.Object)
		return f.body, nil
	}

	return f.forBody(ctx)
}

func (f *FieldReference) processIfFor(ctx context.Context, key string) (Value, bool, error) {
	if ok, err := f.checkCondition(ctx, key); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	body, err := f.getBody(ctx, key)
	if err != nil {
		return nil, false, err
	}
	return f.lookupKeyInValue(ctx, key, body)
}

func (f *FieldReference) cacheValue(key string, v Value) {
	if f.values == nil {
		f.values = map[string]Value{}
	}
	f.values[key] = v
}

func (f *FieldReference) cacheMiss(key string) {
	if f.misses == nil {
		f.misses = map[string]bool{}
	}
	f.misses[key] = true
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
		f.cacheMiss(key)
		return nil, false, nil
	}
	f.cacheValue(key, v)
	return v, true, nil
}

func (f *FieldReference) Keys(ctx context.Context) ([]string, error) {
	if f.keys != nil || f.Field.Key.Match {
		return f.keys, nil
	}

	if f.Field.Embedded {
		embedded, err := f.getEmbeddedValue(ctx)
		if err != nil {
			return nil, err
		}
		embeddedType, err := embedded.Type(ctx)
		if err != nil {
			return nil, err
		}
		if embeddedType == TypeObject {
			return embedded.(ObjectValue).Keys(ctx)
		}
		return []string{EmbeddedKey}, nil
	}

	if f.Field.Key.Name == nil {
		b, err := f.getBody(ctx, "")
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

	s, _, err := f.resolveKey(ctx, "")
	return []string{s}, err
}
