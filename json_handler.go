package flex

import (
	"github.com/goccy/go-json"
	"io"
)

type JsonHandler interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
	MarshalIndent(any, string, string) ([]byte, error)
	NewDecoder(io.Reader) JsonDecoder
	NewEncoder(io.Writer) JsonEncoder
	Validate([]byte) bool
}

type JsonDecoder interface {
	UseNumber()
	DisallowUnknownFields()
	Decode(v any) error
	Buffered() io.Reader
}

type JsonEncoder interface {
	Encode(v any) error
	SetIndent(prefix, indent string)
	SetEscapeHTML(on bool)
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

func (DefaultJsonHandler) NewDecoder(reader io.Reader) JsonDecoder {
	return json.NewDecoder(reader)
}

func (DefaultJsonHandler) NewEncoder(writer io.Writer) JsonEncoder {
	return json.NewEncoder(writer)
}

func (DefaultJsonHandler) Validate(data []byte) bool {
	return json.Valid(data)
}
