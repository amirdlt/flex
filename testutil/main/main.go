package main

import (
	"fmt"
	. "github.com/amirdlt/flex/util"
	"math/rand"
	"sort"
	"time"
)

func main() {
	type T1 struct {
		InT1 struct {
			V6 string
		}
	}

	m := Map[string, any]{"a": "1", "b": "2", "c": "3", "inner": M{"d . m": M{"e": []any{M{"f": "V3"}, "V4", M{"f": (&T1{InT1: struct{ V6 string }{V6: "V6"}})}}}}}
	fmt.Println(m.ValuesByPathForNestedStandardMap("inner.\"d . m\".e.f.InT1", true))

	t := time.Now()

	g := NewGraph[M, M](func(m M) any {
		return m["id"]
	}, func(from *Node[M, M], to *Node[M, M], data M) any {
		arr := []EntityId{from.Id, to.Id}
		sort.Strings(arr)
		return fmt.Sprint(arr[0], arr[1])
	})

	for index := 0; index < 20_000; index++ {
		idF := g.AddNode(M{"id": rand.Int() % 256})
		idT := g.AddNode(M{"id": rand.Int() % 256})

		g.AddEdge(idF, idT, nil)
	}

	fmt.Println(g)
	fmt.Println(time.Since(t))

	//count := Map[string, int]{}
	//g.ForEachNode(func(node *Node[M, M]) {
	//
	//})

	//fmt.Println(count.Values().Sort(func(v1, v2 int) bool {
	//	return v1 > v2
	//})[:10])

	//for k, v := range g.BFSAll() {
	//	fmt.Println(k, v)
	//}
}
