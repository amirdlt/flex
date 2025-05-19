package util

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/k0kubun/pp"
	"github.com/mitchellh/hashstructure/v2"
	"io"
	"os"
	"reflect"
)

type HashOptions = hashstructure.HashOptions

func GenerateUUID(prefix string, v any) string {
	if v == nil {
		if prefix == "" {
			return uuid.New().String()
		}

		return fmt.Sprint(prefix, "--", uuid.New())
	}

	h, err := Hash(v, HashOptions{IgnoreZeroValue: true, SlicesAsSets: true})
	if err != nil {
		panic(err)
	}

	if prefix == "" {
		return uuid.NewHash(
			sha256.New(),
			[16]byte{},
			[]byte(fmt.Sprint(h)),
			4,
		).String()
	}

	return fmt.Sprint(prefix, "--", uuid.NewHash(
		sha256.New(),
		[16]byte{},
		[]byte(fmt.Sprint(h)),
		4,
	))
}

func IsEmpty(object any) bool {

	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
		// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return IsEmpty(deref)
		// for all other types, compare against the zero value
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

func Sha256(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func Sha512(value string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(value)))
}

func Md5(value string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(value)))
}

func Sha256OfHash(value any, options ...HashOptions) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprint(Hash(value, options...)))))
}

func Sha512OfHash(value any, options ...HashOptions) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(fmt.Sprint(Hash(value, options...)))))
}

func Md5OfHash(value any, options ...HashOptions) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprint(Hash(value, options...)))))
}

func Hash(v any, options ...HashOptions) (uint64, error) {
	var opts *HashOptions
	if len(options) > 0 {
		opts = &options[0]
	}

	return hashstructure.Hash(v, hashstructure.FormatV2, opts)
}

func GetFileStream(path string) (io.ReadWriteCloser, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDONLY|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}

	return f, nil
}

func Jsonify(v any) string {
	switch v.(type) {
	case string:
		return v.(string)
	case []byte:
		return string(v.([]byte))
	default:
		v, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			panic(err.Error())
		}
		return string(v)
	}
}

func PrintJsonify(v any, outputs ...io.Writer) {
	if len(outputs) == 0 {
		outputs = []io.Writer{os.Stdout}
	}

	for _, output := range outputs {
		_, _ = fmt.Fprintln(output, Jsonify(v))
	}
}

func Print(v ...any) {
	_, _ = pp.Print(v...)
}

func Printf(format string, v ...any) {
	_, _ = pp.Printf(format, v...)
}

func Sprint(v ...any) string {
	return pp.Sprint(v...)
}

func Sprintln(v ...any) string {
	return pp.Sprintln(v...)
}

func Println(v ...any) {
	_, _ = pp.Println(v...)
}

func Sprintf(format string, v ...any) string {
	return pp.Sprintf(format, v...)
}
