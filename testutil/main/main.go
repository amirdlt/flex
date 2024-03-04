package main

import (
	"fmt"
	. "github.com/amirdlt/flex/util"
)

func main() {
	m := Map[string, any]{"a": "1", "b": "2", "c": "3", "inner": M{"d . m": M{"e": []any{M{"f": "V3"}, "V4"}}}}

	fmt.Println(m.ValuesByPathForNestedStandardMap("inner.\"d . m\".e"))
}
