package flex

import (
	"context"
	"fmt"
	. "github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
)

type NoBody struct{}

var noBody NoBody

type Injector interface {
	Wrap(response any, statusCode int, extValues ...any) Result
	Context() context.Context
	AddResponseHeader(key, value string)
	SetResponseHeader(key, value string)
	ResponseHeaders() http.Header
	URL() *url.URL
	RequestHeaders() http.Header
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
	RequestBody() any
	WrapTextPlain(response any, statusCode int, extValues ...any) Result
	WrapWithContentType(response any, statusCode int, contentType string, extValues ...any) Result
	PathParameter(key string) string
	SetValue(key string, value any)
	Value(key string) any
	LookupValue(key string) (any, bool)
	SetContentType(contentType string)
	DataMap() Map[string, any]
	RemoteAddr() string
	ParseForm() error
	ParseMultipartForm(maxMemory int64) error
	FormValue(key string) string
	PostFormValue(key string) string
	FormFile(key string) (multipart.File, *multipart.FileHeader, error)
	Logger() *log.Logger
	RawPath() string
	Path() string
	request() *http.Request
	response() http.ResponseWriter
}

type BasicInjector struct {
	pathParameters    httprouter.Params
	r                 *http.Request
	w                 http.ResponseWriter
	requestBody       any
	extInjections     Map[string, any]
	defaultErrorCodes Map[int, string]
	rawPath           string
	logger            *log.Logger
}

func (s *BasicInjector) PathParameter(key string) string {
	return s.pathParameters.ByName(key)
}

func (s *BasicInjector) RequestBody() any {
	return s.requestBody
}

func (s *BasicInjector) Context() context.Context {
	return s.r.Context()
}

func (s *BasicInjector) AddResponseHeader(key, value string) {
	s.w.Header().Add(key, value)
}

func (s *BasicInjector) SetResponseHeader(key, value string) {
	s.w.Header().Set(key, value)
}

func (s *BasicInjector) ResponseHeaders() http.Header {
	return s.w.Header()
}

func (s *BasicInjector) URL() *url.URL {
	return s.r.URL
}

func (s *BasicInjector) RequestHeaders() http.Header {
	return s.r.Header
}

func (s *BasicInjector) Host() string {
	return s.r.Host
}

func (s *BasicInjector) Method() string {
	if s.r.Method == "" {
		return http.MethodGet
	}

	return s.r.Method
}

func (s *BasicInjector) ContentLength() int64 {
	return s.r.ContentLength
}

func (s *BasicInjector) Wrap(response any, statusCode int, extValues ...any) Result {
	return Result{
		responseBody: response,
		statusCode:   statusCode,
		extValue:     extValues,
	}
}

func (s *BasicInjector) WrapWithContentType(response any, statusCode int, contentType string, extValues ...any) Result {
	s.SetResponseHeader("Content-Type", contentType)
	return Result{
		responseBody: response,
		statusCode:   statusCode,
		extValue:     extValues,
	}
}

func (s *BasicInjector) WrapOk(response any, extValues ...any) Result {
	return Result{
		responseBody: response,
		extValue:     extValues,
	}
}

func (s *BasicInjector) WrapNoContent(extValues ...any) Result {
	return s.Wrap(nil, http.StatusNoContent, extValues...)
}

func (s *BasicInjector) WrapJsonErr(_error, code string, statusCode int, extValues ...any) Result {
	return s.WrapWithContentType(M{
		"error": _error,
		"code":  code,
	}, statusCode, "application/json", extValues...)
}

func (s *BasicInjector) WrapInvalidBody(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest, extValues...)
}

func (s *BasicInjector) WrapNotFoundErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusNotFound], http.StatusNotFound, extValues...)
}

func (s *BasicInjector) WrapForbiddenErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusForbidden], http.StatusForbidden, extValues...)
}

func (s *BasicInjector) WrapInternalErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusInternalServerError], http.StatusInternalServerError, extValues...)
}

func (s *BasicInjector) WrapBadRequestErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest, extValues...)
}

func (s *BasicInjector) WrapTooManyRequestsErr(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusTooManyRequests], http.StatusTooManyRequests, extValues...)
}

func (s *BasicInjector) WrapTextPlain(response any, statusCode int, extValues ...any) Result {
	return s.WrapWithContentType(fmt.Sprint(response), statusCode, "text/plain", extValues...)
}

func (s *BasicInjector) SetContentType(contentType string) {
	s.SetResponseHeader("Content-Type", contentType)
}

func (s *BasicInjector) Query(key string) string {
	return s.URL().Query().Get(key)
}

func (s *BasicInjector) DefaultQuery(key, defaultValue string) string {
	if s.URL().Query().Has(key) {
		return s.URL().Query().Get(key)
	}

	return defaultValue
}

func (s *BasicInjector) GetRequestHeader(key string) string {
	return s.RequestHeaders().Get(key)
}

func (s *BasicInjector) SetValue(key string, value any) {
	s.extInjections[key] = value
}

func (s *BasicInjector) Value(key string) any {
	return s.extInjections[key]
}

func (s *BasicInjector) LookupValue(key string) (any, bool) {
	v, exist := s.extInjections[key]
	return v, exist
}

func (s *BasicInjector) DataMap() Map[string, any] {
	return s.extInjections
}

func (s *BasicInjector) RemoteAddr() string {
	return s.r.RemoteAddr
}

func (s *BasicInjector) ParseForm() error {
	return s.r.ParseForm()
}

func (s *BasicInjector) ParseMultipartForm(maxMemory int64) error {
	return s.r.ParseMultipartForm(maxMemory)
}

func (s *BasicInjector) FormValue(key string) string {
	return s.r.FormValue(key)
}

func (s *BasicInjector) PostFormValue(key string) string {
	return s.r.PostFormValue(key)
}

func (s *BasicInjector) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return s.r.FormFile(key)
}

func (s *BasicInjector) Logger() *log.Logger {
	return s.logger
}

func (s *BasicInjector) Path() string {
	return s.r.RequestURI
}

func (s *BasicInjector) RawPath() string {
	return s.rawPath
}

func (s *BasicInjector) response() http.ResponseWriter {
	return s.w
}

func (s *BasicInjector) request() *http.Request {
	return s.r
}
