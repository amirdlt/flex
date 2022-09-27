package flex

import (
	"encoding/json"
	"io"
)

type JsonHandler interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
	MarshalIndent(any, string, string) ([]byte, error)
	NewDecoder(io.Reader) *json.Decoder
	NewEncoder(io.Writer) *json.Encoder
	Validate([]byte) bool
}

type DefaultJsonHandler struct{}

func (DefaultJsonHandler) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (DefaultJsonHandler) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (DefaultJsonHandler) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func (DefaultJsonHandler) NewDecoder(reader io.Reader) *json.Decoder {
	return json.NewDecoder(reader)
}

func (DefaultJsonHandler) NewEncoder(writer io.Writer) *json.Encoder {
	return json.NewEncoder(writer)
}

func (DefaultJsonHandler) Validate(data []byte) bool {
	return json.Valid(data)
}
