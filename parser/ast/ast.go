package ast

import (
	"fmt"
)

type Object struct {
	Position Position
	Fields   []*Field
	LBrace   Token
	RBrace   Token
}

type Key struct {
	Position   Position
	Identifier *Literal
	String     *String
	// If match the token for the right brace
	Match *Token
}

type Field struct {
	Position Position
	Key      Key

	Let      *Token
	Embedded bool
	Value    *Value
	If       *If
	For      *For

	StaticKey   string
	StaticValue interface{}

	Separator Token
	Comma     *Token
}

type Whitespace struct {
	Position Position
	Elements []WhitespaceElement
}

type WhitespaceElement struct {
	Comment *Comment
	Space   *Space
}

type Comment struct {
	Position Position
	String   string
}

type Space struct {
	Position Position
	String   string
}

type Token struct {
	Position   Position
	Value      string
	Whitespace Whitespace
}

type If struct {
	Condition *Expression
	Object    *Object
	Else      *If
}

type For struct {
	Position  Position
	IndexVar  *Literal
	ValueVar  *Literal
	Condition *Expression
	Array     *Expression
	Object    *Object
	LBracket  Token
	RBracket  Token
}

type Position struct {
	Source string
	Line   int
	Col    int
	Offset int
}

func (p Position) String() string {
	if p.Source == "" {
		return fmt.Sprintf("%d:%d:%d", p.Line, p.Col, p.Offset)
	}
	return fmt.Sprintf("%s:%d:%d:%d", p.Source, p.Line, p.Col, p.Offset)
}

func (p Position) IsSet() bool {
	return p.Line != 0 || p.Col != 0 || p.Offset != 0
}

type String struct {
	Position   Position
	Parts      []StringPart
	Multiline  bool
	Whitespace Whitespace
}

type StringPart struct {
	String     *string
	Expression *Expression
}

type Number string

type Bool struct {
	Position   Position
	Value      bool
	Whitespace Whitespace
}

type Null struct {
	Position   Position
	Whitespace Whitespace
}

type Value struct {
	Position          Position
	Array             *Array
	Object            *Object
	String            *String
	Number            *Number
	Bool              *Bool
	Null              *Null
	Expression        *Expression
	ListComprehension *For
}

type Op struct {
	Position Position
	Token    Token
}

type Operator struct {
	Position Position
	Op       *Op
	Selector *Selector
}

type Literal struct {
	Position   Position
	Value      string
	Whitespace Whitespace
}

type Expression struct {
	Position Position
	Selector *Selector
	Operator []*Operator
}

type Lookup struct {
	Position Position
	Literal  *Literal
	Index    *Expression
	Start    *Expression
	End      *Expression
	Call     *Call
}

type Call struct {
	Position   Position
	Positional *Array
	Named      *Object
}

type Selector struct {
	Not      bool
	Position Position
	Value    *Value
	Literal  *Literal
	Parens   *Parens
	Lookup   []*Lookup
}

type Array struct {
	Position Position
	Values   []*Value
}

type Parens struct {
	Position
	Value *Value
}
