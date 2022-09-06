package fmt

import (
	"io"
	"regexp"

	"github.com/acorn-io/aml/parser/ast"
)

var (
	identifiedRegexp = regexp.MustCompile("^[_a-zA-Z][_a-zA-Z0-9]*$")
)

type context struct {
	indent   string
	comments map[int]ast.CommentGroups
	out      *writer
}

type writer struct {
	err error
	out io.Writer
}

func (w *writer) write(s string) {
	if w.err != nil {
		return
	}
	_, err := w.out.Write([]byte(s))
	if err != nil {
		w.err = err
	}
}

func Print(out io.Writer, value ast.Value) error {
	comments := value.Comments
}

func printExpression(c context, expr *ast.Expression) {
	if expr.Selector != nil {
		printSelector(c, expr.Selector)
	}
	for _, op := range expr.Operator {
		c.out.write(" ")
		c.out.write(op.Op.Op)
		c.out.write(" ")
		printSelector(c, op.Selector)
	}
}

func printCall(c context, call ast.Call) {
	c.out.write("(")
	i := 0
	if call.Positional != nil {
		for _, arg := range call.Positional.Values {
			if i > 0 {
				c.out.write(", ")
			}
			printValue(c, *arg)
			i++
		}
	}
	if call.Named != nil {
		for _, field := range call.Named.Fields {
			if i > 0 {
				c.out.write(", ")
			}
			printField(c, *field)
			i++
		}
	}
	c.out.write(")")
}

func printKey(c context, key ast.Key) {
	if key.Name == nil {
		return
	}
	if key.Match {
		c.out.write("[~=")
	}
	if !key.Match && len(key.Name.Parts) == 1 && key.Name.Parts[0].String != nil && identifiedRegexp.MatchString(*key.Name.Parts[0].String) {
		c.out.write(*key.Name.Parts[0].String)
	} else {
		printString(c, *key.Name)
	}
	if key.Match {
		c.out.write("]")
	}
	c.out.write(": ")
}

func printFor(c context, forField ast.For) {
	c.out.write("for ")
	if forField.IndexVar != "" {
		c.out.write(forField.IndexVar)
		c.out.write(", ")
	}
	c.out.write(forField.ValueVar)
	c.out.write(" ")
	printExpression(c, forField.Array)
	c.out.write(" ")
	printObject(c, forField.Object)
	if forField.Condition != nil {
		c.out.write(" if ")
		printExpression(c, forField.Condition)
	}
}

func printIf(c context, ifField ast.If) {
	if ifField.Condition != nil {
		c.out.write("if ")
		printExpression(c, ifField.Condition)
		c.out.write(" ")
	}

	printObject(c, *ifField.Object)

	if ifField.Else != nil {
		c.out.write(" else ")
		printIf(c, *ifField.Else)
	}
}

func printField(c context, field ast.Field) {
	switch {
	case field.If != nil:
		printIf(c, *field.If)
	case field.For != nil:
		printFor(c, *field.For)
	case field.Value != nil:
		if !field.Embedded {
			printKey(c, field.Key)
		}
		printValue(c, *field.Value)
	}
}

func printLookup(c context, lookup ast.Lookup) {
	switch {
	case lookup.Index != nil:
		c.out.write("[")
		printExpression(c, lookup.Index)
		c.out.write("]")
	case lookup.Literal != nil:
		c.out.write(".")
		c.out.write(lookup.Literal.Value)
	case lookup.Call != nil:
		printCall(c, *lookup.Call)
	case lookup.End != nil && lookup.Start != nil:
	}
}

func printSelector(c context, sel *ast.Selector) {
	if sel.Not {
		c.out.write("!")
	}
	switch {
	case sel.Literal != nil:
		c.out.write(sel.Literal.Value)
	case sel.Value != nil:
		printValue(c, *sel.Value)
	case sel.Parens != nil:
		c.out.write("(")
		printValue(c, *sel.Parens.Value)
		c.out.write(")")
	}

	for _, lookup := range sel.Lookup {
		printLookup(c, *lookup)
	}
}

func printString(c context, value ast.String) {
	if value.Multiline {
		c.out.write(`"""`)
	} else {
		c.out.write(`"`)
	}

	for _, part := range value.Parts {
		if part.String != nil {
			c.out.write(*part.String)
		}
		if part.Expression != nil {
			c.out.write("\\(")
			printExpression(c, part.Expression)
			c.out.write(")")
		}
	}

	if value.Multiline {
		c.out.write(`"""`)
	} else {
		c.out.write(`"`)
	}
}

func printArray(c context, array ast.Array) {
	newIndent := c.indent + "\t"

	c.out.write("[")
	if len(array.Values) == 1 {
		printValue(c, *array.Values[0])
	} else {
		for _, val := range array.Values {
			c.out.write("\n")
			c.out.write(newIndent)
			printValue(c, *val)
			c.out.write(",")
		}
		if len(array.Values) > 0 {
			c.out.write("\n")
			c.out.write(c.indent)
		}
	}
	c.out.write("]")

	array.
}

func printValue(c context, value ast.Value) {
	switch {
	case value.Null:
		c.out.write("null")
	case value.String != nil:
		printString(c, *value.String)
	case value.Array != nil:
		printArray(c, *value.Array)
	case value.Object != nil:
		printObject(c, *value.Object)
	case value.Expression != nil:
		printExpression(c, value.Expression)
	case value.Bool != nil:
		printBool(c, *value.Bool)
	case value.Number != nil:
		printNumber(c, *value.Number)
	}
}
