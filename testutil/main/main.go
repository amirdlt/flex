package main

import (
	"fmt"
	. "github.com/amirdlt/flex/util"
)

func main() {
	type T1 struct {
		InT1 struct {
			V6 string
		}
	}

	m := Map[string, any]{"a": "1", "b": "2", "c": "3", "inner": M{"d . m": M{"e": []any{M{"f": "V3"}, "V4", M{"f": (&T1{InT1: struct{ V6 string }{V6: "V6"}})}}}}}
	fmt.Println(m.ValuesByPathForNestedStandardMap("inner.\"d . m\".e.f.InT1", true))
}
