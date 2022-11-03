package util

import (
	"fmt"
	"reflect"
	"sync"
)

type M = map[string]any

type MapInterface[K comparable, V any] interface {
	Len() int
	Keys() Stream[K]
	Values() Stream[V]
	Items() Stream[Item[K, V]]
	IsEmpty() bool
	ContainKey(key K) bool
	ContainValue(value V) bool
	Put(key K, value V)
	PutAll(_m Map[K, V])
	Remove(key K)
	RemoveAll(keys ...K)
	PutItem(item Item[K, V])
	Copy() Map[K, V]
	RemoveEmptyValues()
	ForEach(iterator func(k K, v V))
}

type Map[K comparable, V any] map[K]V

type Item[K comparable, V any] struct {
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
		if reflect.DeepEqual(v, value) {
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

func (m Map[K, V]) RemoveEmptyValues() {
	for k, v := range m {
		if IsEmpty(v) {
			delete(m, k)
		}
	}
}

func (m Map[K, V]) ForEach(iterator func(k K, v V)) {
	for k, v := range m {
		iterator(k, v)
	}
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

type SynchronizedMap[K comparable, V any] struct {
	m Map[K, V]
	*sync.RWMutex
}

func NewSynchronizedMap[K comparable, V any](_ K, _ V) SynchronizedMap[K, V] {
	return SynchronizedMap[K, V]{
		m:       Map[K, V]{},
		RWMutex: &sync.RWMutex{},
	}
}

func (m SynchronizedMap[K, V]) Len() int {
	m.Lock()
	defer m.Unlock()

	return m.m.Len()
}

func (m SynchronizedMap[K, V]) Keys() Stream[K] {
	m.Lock()
	defer m.Unlock()

	return m.m.Keys()
}

func (m SynchronizedMap[K, V]) Values() Stream[V] {
	m.Lock()
	defer m.Unlock()

	return m.m.Values()
}

func (m SynchronizedMap[K, V]) Items() Stream[Item[K, V]] {
	m.Lock()
	defer m.Unlock()

	return m.m.Items()
}

func (m SynchronizedMap[K, V]) IsEmpty() bool {
	m.Lock()
	defer m.Unlock()

	return m.m.IsEmpty()
}

func (m SynchronizedMap[K, V]) ContainKey(key K) bool {
	m.Lock()
	defer m.Unlock()

	return m.m.ContainKey(key)
}

func (m SynchronizedMap[K, V]) ContainValue(value V) bool {
	m.Lock()
	defer m.Unlock()

	return m.m.ContainValue(value)
}

func (m SynchronizedMap[K, V]) Put(key K, value V) {
	m.Lock()
	defer m.Unlock()

	m.m.Put(key, value)
}

func (m SynchronizedMap[K, V]) PutAll(_m map[K]V) {
	m.Lock()
	defer m.Unlock()

	m.m.PutAll(_m)
}

func (m SynchronizedMap[K, V]) Remove(key K) {
	m.Lock()
	defer m.Unlock()

	m.m.Remove(key)
}

func (m SynchronizedMap[K, V]) RemoveAll(keys ...K) {
	m.Lock()
	defer m.Unlock()

	m.m.RemoveAll(keys...)
}

func (m SynchronizedMap[K, V]) PutItem(item Item[K, V]) {
	m.Lock()
	defer m.Unlock()

	m.m.PutItem(item)
}

func (m SynchronizedMap[K, V]) Copy() Map[K, V] {
	m.Lock()
	defer m.Unlock()

	return m.m.Copy()
}

func (m SynchronizedMap[K, V]) RemoveEmptyValues() {
	m.Lock()
	defer m.Unlock()

	m.m.RemoveEmptyValues()
}

func (m SynchronizedMap[K, V]) ForEach(iterator func(k K, v V)) {
	m.Lock()
	defer m.Unlock()

	m.m.ForEach(iterator)
}

func (m SynchronizedMap[K, V]) AsMap() Map[K, V] {
	m.Lock()
	defer m.Unlock()

	return m.m.Copy()
}
