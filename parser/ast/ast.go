package ast

import (
	"encoding/json"
	"fmt"
)

type Visitor[T any] interface {
	Visit(o *T)
}

type GenericVisitor interface {
	Visit(o interface{})
}

type Object struct {
	Position Position
	Fields   []Field
}

func visit[T any](o *T, i interface{}) {
	if v, ok := i.(Visitor[T]); ok {
		v.Visit(o)
	}
	if v, ok := i.(GenericVisitor); ok {
		v.Visit(o)
	}
}

func (o *Object) Accept(v interface{}) {
	if o == nil {
		return
	}
	visit(o, v)
	if v, ok := v.(Visitor[Object]); ok {
		v.Visit(o)
	}
	for i := range o.Fields {
		o.Fields[i].Accept(v)
	}
}

type Field struct {
	Position Position
	Key      string
	Value    *Value
	ForField *ForField
}

func (f *Field) Accept(v interface{}) {
	visit(f, v)
	f.Value.Accept(v)
	f.ForField.Accept(v)
}

type ForField struct {
	Position Position
	Var1     string
	Var2     *string
	List     Value
	Object   Object
}

func (f *ForField) Accept(v interface{}) {
	if f == nil {
		return
	}
	visit(f, v)
	f.List.Accept(v)
	f.Object.Accept(v)
}

type Position struct {
	Line   int
	Col    int
	Offset int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d:%d", p.Line, p.Col, p.Offset)
}

func (p Position) IsSet() bool {
	return p.Line != 0 || p.Col != 0 || p.Offset != 0
}

type Value struct {
	Position   Position
	Array      *Array
	Object     *Object
	String     *string
	Number     *json.Number
	Bool       *bool
	Null       bool
	Expression *Expression
}

type Operator struct {
	Op       string
	Selector Selector
}

func (o *Operator) Accept(v interface{}) {
	if o == nil {
		return
	}
	visit(o, v)
	o.Selector.Accept(v)
}

type Literal struct {
	Position Position
	Value    string
}

func (l *Literal) Accept(v interface{}) {
	if l == nil {
		return
	}
	visit(l, v)
}

type Expression struct {
	Position Position
	Selector Selector
	Operator []Operator
}

func (e *Expression) Accept(v interface{}) {
	if e == nil {
		return
	}
	visit(e, v)
	e.Selector.Accept(v)
	for i := range e.Operator {
		e.Operator[i].Accept(v)
	}
}

type Lookup struct {
	Position Position
	Literal  *Literal
	Number   *json.Number
	Call     *Call
}

func (l *Lookup) Accept(v interface{}) {
	if l == nil {
		return
	}
	visit(l, v)
	l.Literal.Accept(v)
	l.Call.Accept(v)
}

type Selector struct {
	Not      bool
	Position Position
	Value    *Value
	Literal  *Literal
	Parens   *Parens
	Lookup   []Lookup
}

type Call struct {
	Position Position
	Args     []Value
}

func (c *Call) Accept(v interface{}) {
	if c == nil {
		return
	}
	visit(c, v)
	for i := range c.Args {
		c.Args[i].Accept(v)
	}
}

func (s *Selector) Accept(v interface{}) {
	if s == nil {
		return
	}
	visit(s, v)
	s.Value.Accept(v)
	s.Literal.Accept(v)
	s.Parens.Accept(v)
	for i := range s.Lookup {
		s.Lookup[i].Accept(v)
	}
}

func (v *Value) Accept(i interface{}) {
	if v == nil {
		return
	}
	visit(v, i)
	v.Array.Accept(i)
	v.Object.Accept(i)
	v.Expression.Accept(i)
}

type Array struct {
	Position      Position
	Values        []Value
	Comprehension *ArrayComprehension
}

func (a *Array) Accept(v interface{}) {
	if a == nil {
		return
	}

	visit(a, v)
	for i := range a.Values {
		a.Values[i].Accept(v)
	}
	a.Comprehension.Accept(v)
}

type Parens struct {
	Position
	Value Value
}

func (p *Parens) Accept(v interface{}) {
	if p == nil {
		return
	}
	visit(p, v)
	p.Value.Accept(v)
}

type ArrayComprehension struct {
	Position Position
	Var1     string
	Var2     *string
	List     Value
	Object   Value
}

func (a *ArrayComprehension) Accept(v interface{}) {
	if a == nil {
		return
	}
	visit(a, v)
	a.List.Accept(v)
	a.Object.Accept(v)
}
