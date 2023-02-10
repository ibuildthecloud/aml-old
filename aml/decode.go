package aml

import (
	"io"
)

type DecoderOptions struct {
	Files     map[string]string
	Args      map[string]any
	Profiles  []string
	InputName string
}

func (d *DecoderOptions) ApplyTo(opts *DecoderOptions) {
	if d == nil {
		return
	}
	if len(d.Files) > 0 {
		if opts.Files == nil {
			opts.Files = map[string]string{}
		}
		for k, v := range d.Files {
			opts.Files[k] = v
		}
	}
	if len(d.Args) > 0 {
		if opts.Args == nil {
			opts.Args = map[string]any{}
		}
		for k, v := range d.Args {
			opts.Args[k] = v
		}
	}

	opts.Profiles = append(opts.Profiles, d.Profiles...)
}

type Option interface {
	ApplyTo(d *DecoderOptions)
}

func WithFile(name, content string) Option {
	return &DecoderOptions{
		Files: map[string]string{
			name: content,
		},
	}
}

func WithFileReader(name string, content io.Reader) (Option, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}
	return &DecoderOptions{
		Files: map[string]string{
			name: string(data),
		},
	}, nil
}

func WithFiles(files map[string]string) Option {
	return &DecoderOptions{
		Files: files,
	}
}

type Decoder struct {
	opts  *DecoderOptions
	input io.Reader
}

func NewDecoder(input io.Reader, options ...Option) *Decoder {
	opts := &DecoderOptions{}
	for _, opt := range options {
		opt.ApplyTo(opts)
	}
	return &Decoder{
		opts:  opts,
		input: input,
	}
}

func (d *Decoder) Decode(v any) error {

}
