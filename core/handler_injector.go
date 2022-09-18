package core

import (
	"context"
	"fmt"
	. "github.com/amirdlt/flex/common"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/url"
)

type NoBody struct{}

var noBody = NoBody{}

type ServerInjector interface {
	Wrap(response any, statusCode int, extValues ...any) HandlerResult
	Context() context.Context
	AddResponseHeader(key, value string)
	SetResponseHeader(key, value string)
	GetResponseHeader() http.Header
	URL() *url.URL
	Headers() http.Header
	Host() string
	Method() string
	ContentLength() int64
	WrapOk(response any, extValues ...any) HandlerResult
	WrapNoContent(extValues ...any) HandlerResult
	WrapJsonErr(_error, code string, statusCode int, extValues ...any) HandlerResult
	WrapInvalidBody(_error string, extValues ...any) HandlerResult
	WrapNotFoundErr(_error string, extValues ...any) HandlerResult
	WrapForbiddenErr(_error string, extValues ...any) HandlerResult
	WrapInternalErr(_error string, extValues ...any) HandlerResult
	WrapBadRequestErr(_error string, extValues ...any) HandlerResult
	WrapTooManyRequestsErr(_error string, extValues ...any) HandlerResult
	Query(key string) string
	DefaultQuery(key, defaultValue string) string
	request() *http.Request
	response() http.ResponseWriter
}

type ServerBaseInjector[SI ServerInjector] struct {
	owner          *Server[SI]
	pathParameters httprouter.Params
	r              *http.Request
	w              http.ResponseWriter
}

type HandlerInjector[SI ServerInjector, B any] struct {
	requestBody B
	ServerBaseInjector[SI]
	SI *SI
}

func (s ServerBaseInjector[SI]) Server() *Server[SI] {
	return s.owner
}

func (h HandlerInjector[_, B]) RequestBody() B {
	return h.requestBody
}

func (s ServerBaseInjector[_]) Context() context.Context {
	return s.r.Context()
}

func (s ServerBaseInjector[_]) AddResponseHeader(key, value string) {
	s.w.Header().Add(key, value)
}

func (s ServerBaseInjector[_]) SetResponseHeader(key, value string) {
	s.w.Header().Set(key, value)
}

func (s ServerBaseInjector[_]) GetResponseHeader() http.Header {
	return s.w.Header()
}

func (s ServerBaseInjector[_]) URL() *url.URL {
	return s.r.URL
}

func (s ServerBaseInjector[_]) Headers() http.Header {
	return s.r.Header
}

func (s ServerBaseInjector[_]) Host() string {
	return s.r.Host
}

func (s ServerBaseInjector[_]) Method() string {
	if s.r.Method == "" {
		return http.MethodGet
	}

	return s.r.Method
}

func (s ServerBaseInjector[_]) ContentLength() int64 {
	return s.r.ContentLength
}

func (s ServerBaseInjector[_]) Wrap(response any, statusCode int, extValues ...any) HandlerResult {
	return HandlerResult{
		responseBody: response,
		statusCode:   statusCode,
		extValue:     extValues,
	}
}

func (s ServerBaseInjector[_]) WrapWithContentType(response any, statusCode int, contentType string, extValues ...any) HandlerResult {
	s.SetResponseHeader("Content-Type", contentType)
	return HandlerResult{
		responseBody: response,
		statusCode:   statusCode,
		extValue:     extValues,
	}
}

func (s ServerBaseInjector[_]) WrapOk(response any, extValues ...any) HandlerResult {
	return HandlerResult{
		responseBody: response,
		extValue:     extValues,
	}
}

func (s ServerBaseInjector[_]) WrapNoContent(extValues ...any) HandlerResult {
	return s.Wrap(nil, http.StatusNoContent, extValues...)
}

func (s ServerBaseInjector[_]) WrapJsonErr(_error, code string, statusCode int, extValues ...any) HandlerResult {
	return HandlerResult{
		responseBody: M{
			"error": _error,
			"code":  code,
		},
		statusCode: statusCode,
		extValue:   extValues,
	}
}

func (s ServerBaseInjector[_]) WrapInvalidBody(_error string, extValues ...any) HandlerResult {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest, extValues...)
}

func (s ServerBaseInjector[_]) WrapNotFoundErr(_error string, extValues ...any) HandlerResult {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusFound], http.StatusNotFound, extValues...)
}

func (s ServerBaseInjector[_]) WrapForbiddenErr(_error string, extValues ...any) HandlerResult {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusForbidden], http.StatusForbidden, extValues...)
}

func (s ServerBaseInjector[_]) WrapInternalErr(_error string, extValues ...any) HandlerResult {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusInternalServerError], http.StatusInternalServerError, extValues...)
}

func (s ServerBaseInjector[_]) WrapBadRequestErr(_error string, extValues ...any) HandlerResult {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest, extValues...)
}

func (s ServerBaseInjector[_]) WrapTooManyRequestsErr(_error string, extValues ...any) HandlerResult {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusTooManyRequests], http.StatusTooManyRequests, extValues...)
}

func (s ServerBaseInjector[_]) WrapTextPlain(response any, statusCode int, extValues ...any) HandlerResult {
	return s.WrapWithContentType(fmt.Sprint(response), statusCode, "text/plain", extValues...)
}

func (s ServerBaseInjector[_]) SetContentType(contentType string) {
	s.SetResponseHeader("Content-Type", contentType)
}

func (s ServerBaseInjector[_]) Query(key string) string {
	return s.URL().Query().Get(key)
}

func (s ServerBaseInjector[_]) DefaultQuery(key, defaultValue string) string {
	if s.URL().Query().Has(key) {
		return s.URL().Query().Get(key)
	}

	return defaultValue
}

func (s ServerBaseInjector[_]) request() *http.Request {
	return s.r
}

func (s ServerBaseInjector[_]) response() http.ResponseWriter {
	return s.w
}
