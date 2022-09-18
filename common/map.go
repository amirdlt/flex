package common

import "fmt"

type M = map[string]any

type Map[K, V comparable] map[K]V

type Item[K, V comparable] struct {
	k K
	v V
}

func (i Item[K, V]) Key() K {
	return i.k
}

func (i Item[K, V]) Value() V {
	return i.v
}

func (i Item[K, V]) String() string {
	return fmt.Sprintf("[key=%v, value=%v]", i.k, i.v)
}

func (m Map[K, V]) Len() int {
	return len(m)
}

func (m Map[K, V]) Keys() Stream[K] {
	keys := make([]K, len(m))
	index := 0
	for k := range m {
		keys[index] = k
		index++
	}

	return keys
}

func (m Map[K, V]) Values() Stream[V] {
	values := make([]V, len(m))
	index := 0
	for _, v := range m {
		values[index] = v
		index++
	}

	return values
}

func (m Map[K, V]) Items() Stream[Item[K, V]] {
	items := make([]Item[K, V], 0, len(m))
	for k, v := range m {
		items = append(items, Item[K, V]{k, v})
	}

	return items
}

func (m Map[K, V]) IsEmpty() bool {
	return len(m) == 0
}

func (m Map[K, V]) ContainKey(key K) bool {
	_, exist := m[key]
	return exist
}

func (m Map[K, V]) ContainValue(value V) bool {
	for _, v := range m {
		if v == value {
			return true
		}
	}

	return false
}

func (m Map[K, V]) Put(key K, value V) {
	m[key] = value
}

func (m Map[K, V]) PutAll(_m Map[K, V]) {
	for k, v := range _m {
		m[k] = v
	}
}

func (m Map[K, V]) Remove(key K) {
	delete(m, key)
}

func (m Map[K, V]) RemoveAll(keys ...K) {
	for _, key := range keys {
		delete(m, key)
	}
}

func (m Map[K, V]) PutItem(item Item[K, V]) {
	m[item.k] = item.v
}

func (m Map[K, V]) Copy() Map[K, V] {
	res := Map[K, V]{}
	for k, v := range m {
		res[k] = v
	}

	return res
}

func MapOf[K, V comparable](items ...Item[K, V]) Map[K, V] {
	m := Map[K, V]{}
	for _, item := range items {
		m[item.k] = item.v
	}

	return m
}

func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	res := map[K]V{}
	for k, v := range m {
		res[k] = v
	}

	return res
}
