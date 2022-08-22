package aml

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/acorn-io/aml/eval"
	"github.com/acorn-io/aml/parser"
	"github.com/acorn-io/aml/parser/ast"
)

type Decoder struct {
	in       io.Reader
	filename string
	ctx      context.Context
	ticks    int
}

func (d *Decoder) getCtx() context.Context {
	return eval.WithTicks(d.ctx, d.ticks)
}

func (d *Decoder) SetTicks(ticks int) {
	d.ticks = ticks
}

func (d *Decoder) SetContext(ctx context.Context) {
	d.ctx = ctx
}

func (d *Decoder) SetFilename(filename string) {
	d.filename = filename
}

func (d *Decoder) Decode(out interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	return d.decode(out)
}

func (d *Decoder) decode(out any) error {
	output, err := parser.ParseReader(d.filename, d.in)
	if err != nil {
		return err
	}
	if inAST, ok := out.(*ast.Value); ok {
		*inAST = *output.(*ast.Value)
		return nil
	}
	ctx := d.getCtx()
	v, err := eval.ToValue(ctx, &eval.Scope{}, output.(*ast.Value))
	if err != nil {
		return err
	}
	f, err := v.Interface(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		in:    r,
		ctx:   context.Background(),
		ticks: 10000,
	}
}
