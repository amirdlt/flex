package util

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"reflect"
)

// IsEmpty gets whether the specified object is considered empty or not.
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

func Md5(value string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(value)))
}
