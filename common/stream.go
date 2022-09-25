package common

import (
	"reflect"
	"sort"
)

type Stream[V any] []V

func (s Stream[V]) Len() int {
	return len(s)
}

func (s Stream[V]) ForEach(consumer func(i int, v V)) {
	for index, v := range s {
		consumer(index, v)
	}
}

func (s Stream[V]) AddAll(stream Stream[V]) Stream[V] {
	return append(s, stream...)
}

func (s Stream[V]) Append(v ...V) Stream[V] {
	return append(s, v...)
}

func (s Stream[V]) AppendIf(predicate func(i int, v V) bool, v ...V) Stream[V] {
	res := s
	for i, v := range v {
		if predicate(i, v) {
			res = append(s, v)
		}
	}

	return res
}

func (s Stream[V]) Sort(less func(v1, v2 V) bool) Stream[V] {
	sort.Slice(s, func(i, j int) bool {
		return less(s[i], s[j])
	})

	return s
}

func (s Stream[V]) RemoveIndex(index int) Stream[V] {
	return append(s[:index], s[index+1:]...)
}

func (s Stream[V]) Remove(v V) Stream[V] {
	res := make(Stream[V], 0, len(s))
	for _, vSrc := range s {
		if !reflect.DeepEqual(vSrc, v) {
			res = append(res, vSrc)
		}
	}

	return res
}

func (s Stream[V]) RemoveAll(v Stream[V]) Stream[V] {
	_map := map[reflect.Value]any{}
	for _, v := range v {
		_map[reflect.ValueOf(v)] = nil
	}

	res := make(Stream[V], 0, len(s))
	for _, v := range s {
		if _, exist := _map[reflect.ValueOf(v)]; !exist {
			res = append(res, v)
		}
	}

	return res
}

func (s Stream[V]) RemoveIf(predicate func(i int, v V) bool) Stream[V] {
	res := make(Stream[V], 0, len(s))

	for i, v := range s {
		if !predicate(i, v) {
			res = append(res, v)
		}
	}

	return res
}

func (s Stream[V]) Filter(filter func(i int, v V) bool) Stream[V] {
	return s.RemoveIf(filter)
}

func (s Stream[V]) RemoveEmptyValues() Stream[V] {
	return s.RemoveIf(func(_ int, v V) bool {
		return IsEmpty(v)
	})
}

func (s Stream[V]) MapToInt(mapper func(i int, v V) int) Stream[int] {
	return MapStream(s, mapper)
}

func (s Stream[V]) Sum(sum func(v1, v2 V) V) V {
	var res V
	if len(s) == 0 {
		return res
	}

	res = s[0]
	for _, v := range s[1:] {
		res = sum(res, v)
	}

	return res
}

func (s Stream[V]) MapToFloat(mapper func(i int, v V) float64) Stream[float64] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToString(mapper func(i int, v V) string) Stream[string] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToByte(mapper func(i int, v V) byte) Stream[byte] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToUint(mapper func(i int, v V) uint) Stream[uint] {
	return MapStream(s, mapper)
}

func (s Stream[V]) Contains(v V) bool {
	for _, sv := range s {
		if reflect.DeepEqual(v, sv) {
			return true
		}
	}

	return false
}

func (s Stream[V]) ContainsAll(v Stream[V]) bool {
	for _, dv := range v {
		if !s.Contains(dv) {
			return false
		}
	}

	return true
}

func (s Stream[V]) Limit(n int) Stream[V] {
	if n < 0 {
		panic("n must not be negative")
	}

	_len := len(s)
	if n > _len {
		n = _len
	}

	return s[:n]
}

func (s Stream[V]) Skip(n int) Stream[V] {
	if n < 0 {
		panic("n must not be negative")
	}

	_len := len(s)
	if n > _len {
		n = _len
	}

	return s[n:]
}

func (s Stream[V]) Distinct() Stream[V] {
	m := map[reflect.Value]any{}
	for _, v := range s {
		m[reflect.ValueOf(v)] = nil
	}

	res := make(Stream[V], 0, len(s))
	for k := range m {
		res = append(res, k.Interface().(V))
	}

	return res
}

func (s Stream[V]) AllMatch(predicate func(i int, v V) bool) bool {
	for i, v := range s {
		if !predicate(i, v) {
			return false
		}
	}

	return true
}

func (s Stream[V]) AnyMatch(predicate func(i int, v V) bool) bool {
	for i, v := range s {
		if predicate(i, v) {
			return true
		}
	}

	return false
}

func (s Stream[V]) NoneMatch(predicate func(i int, v V) bool) bool {
	for i, v := range s {
		if predicate(i, v) {
			return false
		}
	}

	return true
}

func (s Stream[V]) IsEmpty() bool {
	return len(s) == 0
}

func (s Stream[V]) Copy() Stream[V] {
	res := make(Stream[V], len(s))
	for i, v := range s {
		res[i] = v
	}

	return res
}

func MapStream[F any, T any](source Stream[F], mapper func(i int, f F) T) Stream[T] {
	res := make(Stream[T], len(source))
	for i, v := range source {
		res[i] = mapper(i, v)
	}

	return res
}

func StreamOf[V any](source ...V) Stream[V] {
	return source
}

func CopySlice[V any](src []V) []V {
	clone := make([]V, len(src))
	for i, v := range src {
		clone[i] = v
	}

	return clone
}
