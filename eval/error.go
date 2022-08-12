package eval

import (
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type ErrPositionWrapped struct {
	Pos ast.Position
	Err error
}

func (e *ErrPositionWrapped) Error() string {
	return fmt.Sprintf("%s: %v", e.Pos, e.Err)
}

func (e *ErrPositionWrapped) Unwrap() error {
	return e.Err
}

func wrapErr(pos ast.Position, err error) error {
	if err == nil {
		return nil
	}
	return &ErrPositionWrapped{
		Pos: pos,
		Err: err,
	}
}
