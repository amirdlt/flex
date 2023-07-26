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
	PutAll(mm map[K]V)
	Remove(key K)
	RemoveAll(keys ...K)
	PutItems(items ...Item[K, V])
	Copy() MapInterface[K, V]
	RemoveEmptyValues()
	ForEach(iterator func(k K, v V))
	ItemSample(size int, allowRepeating bool) Stream[Item[K, V]]
	Sample(size int, allowRepeating bool) MapInterface[K, V]
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

func (m Map[K, V]) PutAll(mm map[K]V) {
	for k, v := range mm {
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

func (m Map[K, V]) PutItems(items ...Item[K, V]) {
	for _, item := range items {
		m[item.k] = item.v
	}
}

func (m Map[K, V]) Copy() MapInterface[K, V] {
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

func (m Map[K, V]) ItemSample(size int, allowRepeating bool) Stream[Item[K, V]] {
	if size == 0 {
		return []Item[K, V]{}
	}

	if size < 0 || size > m.Len() {
		size = m.Len()
	}

	return m.Items().Sample(size, allowRepeating)
}

func (m Map[K, V]) Sample(size int, allowRepeating bool) MapInterface[K, V] {
	if size == 0 {
		return Map[K, V]{}
	}

	if size < 0 || size > m.Len() {
		size = m.Len()
	}

	return MapOf([]Item[K, V](m.ItemSample(size, allowRepeating))...)
}

func MapOf[K comparable, V any](items ...Item[K, V]) Map[K, V] {
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
	m MapInterface[K, V]
	*sync.RWMutex
}

func NewSynchronizedMap[K comparable, V any](m map[K]V) SynchronizedMap[K, V] {
	return SynchronizedMap[K, V]{
		m:       Map[K, V](m),
		RWMutex: &sync.RWMutex{},
	}
}

func (m SynchronizedMap[K, V]) Len() int {
	m.RLock()
	defer m.RUnlock()

	return m.m.Len()
}

func (m SynchronizedMap[K, V]) Keys() Stream[K] {
	m.RLock()
	defer m.RUnlock()

	return m.m.Keys()
}

func (m SynchronizedMap[K, V]) Values() Stream[V] {
	m.RLock()
	defer m.RUnlock()

	return m.m.Values()
}

func (m SynchronizedMap[K, V]) Items() Stream[Item[K, V]] {
	m.RLock()
	defer m.RUnlock()

	return m.m.Items()
}

func (m SynchronizedMap[K, V]) IsEmpty() bool {
	m.RLock()
	defer m.RUnlock()

	return m.m.IsEmpty()
}

func (m SynchronizedMap[K, V]) ContainKey(key K) bool {
	m.RLock()
	defer m.RUnlock()

	return m.m.ContainKey(key)
}

func (m SynchronizedMap[K, V]) ContainValue(value V) bool {
	m.RLock()
	defer m.RUnlock()

	return m.m.ContainValue(value)
}

func (m SynchronizedMap[K, V]) Put(key K, value V) {
	m.Lock()
	defer m.Unlock()

	m.m.Put(key, value)
}

func (m SynchronizedMap[K, V]) PutAll(mm map[K]V) {
	m.Lock()
	defer m.Unlock()

	m.m.PutAll(mm)
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

func (m SynchronizedMap[K, V]) PutItems(items ...Item[K, V]) {
	m.Lock()
	defer m.Unlock()

	m.m.PutItems(items...)
}

func (m SynchronizedMap[K, V]) Copy() MapInterface[K, V] {
	m.RLock()
	defer m.RUnlock()

	return SynchronizedMap[K, V]{
		m:       m.m.Copy(),
		RWMutex: &sync.RWMutex{},
	}
}

func (m SynchronizedMap[K, V]) RemoveEmptyValues() {
	m.Lock()
	defer m.Unlock()

	m.m.RemoveEmptyValues()
}

func (m SynchronizedMap[K, V]) ForEach(iterator func(k K, v V)) {
	m.RLock()
	defer m.RUnlock()

	m.m.ForEach(iterator)
}

func (m SynchronizedMap[K, V]) ItemSample(size int, allowRepeating bool) Stream[Item[K, V]] {
	m.RLock()
	defer m.RUnlock()

	return m.m.ItemSample(size, allowRepeating)
}

func (m SynchronizedMap[K, V]) Sample(size int, allowRepeating bool) MapInterface[K, V] {
	m.RLock()
	defer m.RUnlock()

	return m.m.Sample(size, allowRepeating)
}

func (m SynchronizedMap[K, V]) UnsafeInnerMap() MapInterface[K, V] {
	return m.m
}
