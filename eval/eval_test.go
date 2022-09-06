package eval

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acorn-io/aml/parser"
	"github.com/acorn-io/aml/parser/ast"
	"github.com/stretchr/testify/assert"
)

func TestEval(t *testing.T) {
	const dir = "./testdata/eval"
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		t.Run(strings.TrimSuffix(file.Name(), ".test"), func(t *testing.T) {
			bytes, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			expected := string(bytes)
			i := strings.Index(expected, "---")
			input := expected[:i]
			expected = expected[i+4:]

			node, err := parser.Parse(file.Name(), []byte(input), parser.GlobalStore("source", file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithTicks(context.Background(), 10000)
			v, err := Eval(ctx, &Scope{}, node.(*ast.Value))
			if err != nil {
				t.Fatal(err)
			}
			iface, err := v.Interface(ctx)
			if err != nil {
				t.Fatal(err)
			}
			result, err := json.MarshalIndent(iface, "", "    ")
			if err != nil {
				t.Fatal(err)
			}

			expected, err = normalizeJSON(expected)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, expected, string(result))
		})
	}
}

func normalizeJSON(expected string) (string, error) {
	obj := map[string]any{}
	err := json.Unmarshal([]byte(strings.TrimSpace(expected)), &obj)
	if err != nil {
		return "", err
	}
	res, err := json.MarshalIndent(obj, "", "    ")
	return string(res), err
}
