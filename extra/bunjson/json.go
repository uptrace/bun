package bunjson

import (
	"encoding/json"
	"io"
)

var _ Provider = (*StdProvider)(nil)

type StdProvider struct{}

func (StdProvider) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (StdProvider) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (StdProvider) NewEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

func (StdProvider) NewDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
