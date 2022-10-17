package flex

import "reflect"

type Result struct {
	responseBody any
	statusCode   int
	extValue     any
	terminate    bool
}

func (h Result) GetExtValue() any {
	if h.extValue == nil {
		return nil
	}

	val := reflect.ValueOf(h.extValue)
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		if val.Len() == 1 {
			return val.Slice(0, 1)
		}
	}

	return h.extValue
}
