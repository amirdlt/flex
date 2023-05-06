package flex

import "reflect"

type Result struct {
	responseBody any
	statusCode   int
	extValue     any
	terminate    bool
}

func (r Result) GetExtValue() any {
	if r.extValue == nil {
		return nil
	}

	val := reflect.ValueOf(r.extValue)
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		if val.Len() == 1 {
			return val.Slice(0, 1)
		}
	}

	return r.extValue
}

func (r Result) IsSuccessful() bool {
	return r.statusCode != 0 && r.statusCode < 300 && r.statusCode >= 200
}

func (r Result) IsTerminated() bool {
	return r.terminate
}
