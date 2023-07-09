package flex

import (
	"net/http"
)

type Result struct {
	responseBody any
	statusCode   int
	terminate    bool
}

func (r Result) IsSuccessful() bool {
	return r.statusCode == 0 || r.statusCode < 300 && r.statusCode >= 200
}

func (r Result) IsTerminated() bool {
	return r.terminate
}

func (r Result) Body() any {
	return r.responseBody
}

func (r Result) StatusCode() int {
	if r.statusCode == 0 {
		return http.StatusOK
	}

	return r.statusCode
}
