package flx

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

type Injector interface {
	Wrap(response any, statusCode int, extValues ...any) Result
	Context() context.Context
	AddResponseHeader(key, value string)
	SetResponseHeader(key, value string)
	GetResponseHeaders() http.Header
	URL() *url.URL
	GetRequestHeaders() http.Header
	GetRequestHeader(key string) string
	Host() string
	Method() string
	ContentLength() int64
	WrapOk(response any, extValues ...any) Result
	WrapNoContent(extValues ...any) Result
	WrapJsonErr(_error, code string, statusCode int, extValues ...any) Result
	WrapInvalidBody(_error string, extValues ...any) Result
	WrapNotFoundErr(_error string, extValues ...any) Result
	WrapForbiddenErr(_error string, extValues ...any) Result
	WrapInternalErr(_error string, extValues ...any) Result
	WrapBadRequestErr(_error string, extValues ...any) Result
	WrapTooManyRequestsErr(_error string, extValues ...any) Result
	Query(key string) string
	DefaultQuery(key, defaultValue string) string
	request() *http.Request
	response() http.ResponseWriter
	RequestBody() any
	WrapTextPlain(response any, statusCode int, extValues ...any) Result
	WrapWithContentType(response any, statusCode int, contentType string, extValues ...any) Result
	PathParameter(key string) string
	SetValue(key string, value any)
	GetValue(key string) any
	LookupValue(key string) (any, bool)
	SetContentType(contentType string)
	GetDataMap() Map[string, any]
	RemoteAddr() string
}

type BasicInjector[I Injector] struct {
	owner          *Server[I]
	pathParameters httprouter.Params
	r              *http.Request
	w              http.ResponseWriter
	requestBody    any
	extInjections  Map[string, any]
}

func (s *BasicInjector[I]) Server() *Server[I] {
	return s.owner
}

func (s *BasicInjector[_]) PathParameter(key string) string {
	return s.pathParameters.ByName(key)
}

func (s *BasicInjector[_]) RequestBody() any {
	return s.requestBody
}

func (s *BasicInjector[_]) Context() context.Context {
	return s.r.Context()
}

func (s *BasicInjector[_]) AddResponseHeader(key, value string) {
	s.w.Header().Add(key, value)
}

func (s *BasicInjector[_]) SetResponseHeader(key, value string) {
	s.w.Header().Set(key, value)
}

func (s *BasicInjector[_]) GetResponseHeaders() http.Header {
	return s.w.Header()
}

func (s *BasicInjector[_]) URL() *url.URL {
	return s.r.URL
}

func (s *BasicInjector[_]) GetRequestHeaders() http.Header {
	return s.r.Header
}

func (s *BasicInjector[_]) Host() string {
	return s.r.Host
}

func (s *BasicInjector[_]) Method() string {
	if s.r.Method == "" {
		return http.MethodGet
	}

	return s.r.Method
}

func (s *BasicInjector[_]) ContentLength() int64 {
	return s.r.ContentLength
}

func (s *BasicInjector[_]) Wrap(response any, statusCode int, extValues ...any) Result {
	return Result{
		responseBody: response,
		statusCode:   statusCode,
		extValue:     extValues,
	}
}

func (s *BasicInjector[_]) WrapWithContentType(response any, statusCode int, contentType string, extValues ...any) Result {
	s.SetResponseHeader("Content-Type", contentType)
	return Result{
		responseBody: response,
		statusCode:   statusCode,
		extValue:     extValues,
	}
}

func (s *BasicInjector[_]) WrapOk(response any, extValues ...any) Result {
	return Result{
		responseBody: response,
		extValue:     extValues,
	}
}

func (s *BasicInjector[_]) WrapNoContent(extValues ...any) Result {
	return s.Wrap(nil, http.StatusNoContent, extValues...)
}

func (s *BasicInjector[_]) WrapJsonErr(_error, code string, statusCode int, extValues ...any) Result {
	return s.WrapWithContentType(M{
		"error": _error,
		"code":  code,
	}, statusCode, "application/json", extValues...)
}

func (s *BasicInjector[_]) WrapInvalidBody(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest, extValues...)
}

func (s *BasicInjector[_]) WrapNotFoundErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusFound], http.StatusNotFound, extValues...)
}

func (s *BasicInjector[_]) WrapForbiddenErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusForbidden], http.StatusForbidden, extValues...)
}

func (s *BasicInjector[_]) WrapInternalErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusInternalServerError], http.StatusInternalServerError, extValues...)
}

func (s *BasicInjector[_]) WrapBadRequestErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest, extValues...)
}

func (s *BasicInjector[_]) WrapTooManyRequestsErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.owner.defaultErrorCodes[http.StatusTooManyRequests], http.StatusTooManyRequests, extValues...)
}

func (s *BasicInjector[_]) WrapTextPlain(response any, statusCode int, extValues ...any) Result {
	return s.WrapWithContentType(fmt.Sprint(response), statusCode, "text/plain", extValues...)
}

func (s *BasicInjector[_]) SetContentType(contentType string) {
	s.SetResponseHeader("Content-Type", contentType)
}

func (s *BasicInjector[_]) Query(key string) string {
	return s.URL().Query().Get(key)
}

func (s *BasicInjector[_]) DefaultQuery(key, defaultValue string) string {
	if s.URL().Query().Has(key) {
		return s.URL().Query().Get(key)
	}

	return defaultValue
}

func (s *BasicInjector[_]) request() *http.Request {
	return s.r
}

func (s *BasicInjector[_]) response() http.ResponseWriter {
	return s.w
}

func (s *BasicInjector[_]) GetRequestHeader(key string) string {
	return s.GetRequestHeaders().Get(key)
}

func (s *BasicInjector[_]) SetValue(key string, value any) {
	s.extInjections[key] = value
}

func (s *BasicInjector[_]) GetValue(key string) any {
	return s.extInjections[key]
}

func (s *BasicInjector[_]) LookupValue(key string) (any, bool) {
	v, exist := s.extInjections[key]
	return v, exist
}

func (s *BasicInjector[_]) GetDataMap() Map[string, any] {
	return s.extInjections
}

func (s *BasicInjector[_]) RemoteAddr() string {
	return s.r.RemoteAddr
}

func (s *BasicInjector[_]) Forms() {

}
