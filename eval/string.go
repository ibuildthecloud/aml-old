package eval

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/acorn-io/aml/parser/ast"
)

type ItCouldBe string

const (
	True  ItCouldBe = "true"
	False ItCouldBe = "false"
	Maybe ItCouldBe = "maybe"
)

func trimIndent(str string) string {
	scanner := bufio.NewScanner(strings.NewReader(str))
	if !scanner.Scan() || scanner.Text() != "" {
		return str
	}

	if !scanner.Scan() {
		return str
	}
	line := scanner.Text()
	prefix := ""
	for _, c := range []byte{'\t', ' '} {
		if prefix == "" {
			for i := 0; i < len(line); i++ {
				if line[i] == c {
					prefix += string(c)
				} else {
					break
				}
			}
		}
	}

	if prefix == "" {
		return str
	}

	result := &strings.Builder{}
	result.WriteString(strings.TrimPrefix(line, prefix))
	for scanner.Scan() {
		result.WriteString("\n")
		result.WriteString(strings.TrimPrefix(scanner.Text(), prefix))
	}

	return result.String()
}

func EvaluateString(ctx context.Context, scope *Scope, s *ast.String) (_ string, err error) {
	defer func() {
		err = wrapErr(s.Position, err)
	}()
	buf := &strings.Builder{}
	for _, part := range s.Parts {
		if part.String != nil {
			buf.WriteString(*part.String)
		} else if part.Expression != nil {
			v, err := EvaluateExpression(ctx, scope, part.Expression)
			if err != nil {
				return "", err
			}
			iface, err := v.Interface(ctx)
			if err != nil {
				return "", err
			}
			buf.WriteString(fmt.Sprint(iface))
		}
	}

	return trimIndent(buf.String()), nil
}

func QuickMatch(s *ast.String, val string) ItCouldBe {
	if len(s.Parts) == 0 && val == "" {
		return True
	} else if len(s.Parts) == 1 && s.Parts[0].String != nil {
		if *s.Parts[0].String == val {
			return True
		}
		return False
	}

	if stringToPattern(s).MatchString(val) {
		return Maybe
	}
	return False
}

func stringToPattern(s *ast.String) *regexp.Regexp {
	buf := &strings.Builder{}
	buf.WriteString("^")
	for _, part := range s.Parts {
		if part.String != nil {
			buf.WriteString(regexp.QuoteMeta(*part.String))
		} else if part.Expression != nil {
			buf.WriteString(".*")
		}
	}
	buf.WriteString("$")
	return regexp.MustCompile(buf.String())
}
