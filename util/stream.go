package util

import (
	"github.com/pkg/errors"
	"math/rand"
	"reflect"
	"sort"
)

type Stream[V any] []V

func (s Stream[V]) Len() int {
	return len(s)
}

func (s Stream[V]) ForEach(consumer func(v V)) {
	for _, v := range s {
		consumer(v)
	}
}

func (s Stream[V]) ForEachWithIndexedConsumer(consumer func(i int, v V)) {
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

func (s Stream[V]) AppendIf(predicate func(v V) bool, v ...V) Stream[V] {
	res := s
	for _, v := range v {
		if predicate(v) {
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

func (s Stream[V]) RemoveIf(predicate func(v V) bool) Stream[V] {
	res := make(Stream[V], 0, len(s))

	for _, v := range s {
		if !predicate(v) {
			res = append(res, v)
		}
	}

	return res
}

func (s Stream[V]) Filter(filter func(v V) bool) Stream[V] {
	return s.RemoveIf(filter)
}

func (s Stream[V]) RemoveEmptyValues() Stream[V] {
	return s.RemoveIf(func(v V) bool {
		return IsEmpty(v)
	})
}

func (s Stream[V]) MapToInt(mapper func(v V) int) Stream[int] {
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

func (s Stream[V]) MapToFloat(mapper func(v V) float64) Stream[float64] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToString(mapper func(v V) string) Stream[string] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToByte(mapper func(v V) byte) Stream[byte] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToUint(mapper func(v V) uint) Stream[uint] {
	return MapStream(s, mapper)
}

func (s Stream[V]) MapToJson(mapper func(v V) M) Stream[M] {
	res := make([]M, len(s))
	for i, v := range s {
		res[i] = mapper(v)
	}

	return res
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
	var distinct []V
	checkDuplicationMap := map[uint64]any{}
	for _, v := range s {
		hash, _ := Hash(v)
		if _, duplicated := checkDuplicationMap[hash]; duplicated {
			continue
		}

		distinct = append(distinct, v)
		checkDuplicationMap[hash] = nil
	}

	return distinct
}

func (s Stream[V]) AllMatch(predicate func(v V) bool) bool {
	for _, v := range s {
		if !predicate(v) {
			return false
		}
	}

	return true
}

func (s Stream[V]) AnyMatch(predicate func(v V) bool) bool {
	for _, v := range s {
		if predicate(v) {
			return true
		}
	}

	return false
}

func (s Stream[V]) NoneMatch(predicate func(v V) bool) bool {
	for _, v := range s {
		if predicate(v) {
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
	copy(res, s)

	return res
}

func (s Stream[V]) Sample(size int, allowRepeating bool) Stream[V] {
	if size == 0 {
		return []V{}
	}

	if size < 0 || size > len(s) {
		size = len(s)
	}

	result := make(Stream[V], size)
	if allowRepeating {
		for i := 0; i < size; i++ {
			result[i] = s[rand.Intn(len(s))]
		}

		return result
	}

	chosenIndices := Map[int, any]{}
	for i := 0; i < size; i++ {
		randomIndex := rand.Intn(len(s))
		if chosenIndices.ContainKey(randomIndex) {
			i--
			continue
		}

		result[i] = s[randomIndex]
		chosenIndices[randomIndex] = nil
	}

	return result
}

func (s Stream[V]) Find(filter func(v V) bool) Stream[V] {
	var res []V
	for _, v := range s {
		if filter(v) {
			res = append(res, v)
		}
	}

	return res
}

func (s Stream[V]) FindOne(filter func(v V) bool) (V, error) {
	for _, v := range s {
		if filter(v) {
			return v, nil
		}
	}

	var v V
	return v, errors.New("element does not exist")
}

func (s Stream[V]) Update(updater func(v V) V) Stream[V] {
	for i, v := range s {
		s[i] = updater(v)
	}

	return s
}

func (s Stream[V]) UpdateIf(predicate func(v V) bool, updater func(v V) V) Stream[V] {
	for i, v := range s {
		if predicate(v) {
			s[i] = updater(v)
		}
	}

	return s
}

func (s Stream[V]) UpdateOneIf(predicate func(v V) bool, updater func(v V) V) Stream[V] {
	for i, v := range s {
		if predicate(v) {
			s[i] = updater(v)
			return s
		}
	}

	return s
}

func (s Stream[V]) Group(labelGenerator func(v V) string) map[string]Stream[V] {
	result := map[string]Stream[V]{}
	for _, v := range s {
		label := labelGenerator(v)
		result[label] = append(result[label], v)
	}

	return result
}

func (s Stream[V]) Reduce(reducer func(v1, v2 V) V) V {
	if len(s) == 0 {
		panic("empty stream does not support reduce method")
	}

	reduced := s[0]
	for i := 1; i < len(s); i++ {
		reduced = reducer(reduced, s[i])
	}

	return reduced
}

func (s Stream[V]) AppendIfNotExist(vs ...V) Stream[V] {
	result := s
	for _, v := range vs {
		if !result.Contains(v) {
			result = append(result, v)
		}
	}

	return result
}

func (s Stream[V]) AppendIfNotExistAndNotEmpty(vs ...V) Stream[V] {
	result := s
	for _, v := range vs {
		if !IsEmpty(v) && !result.Contains(v) {
			result = append(result, v)
		}
	}

	return result
}

func (s Stream[V]) AppendIfNotEmpty(vs ...V) Stream[V] {
	result := s
	for _, v := range vs {
		if !IsEmpty(v) {
			result = append(result, v)
		}
	}

	return result
}

func MapStream[F any, T any](source Stream[F], mapper func(f F) T) Stream[T] {
	res := make(Stream[T], len(source))
	for i, v := range source {
		res[i] = mapper(v)
	}

	return res
}

func StreamOf[V any](source ...V) Stream[V] {
	return source
}

func CopySlice[V any](src []V) []V {
	clone := make([]V, len(src))
	copy(clone, src)

	return clone
}
