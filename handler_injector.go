package flex

import (
	"context"
	"fmt"
	. "github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	FormFile(name string) (*multipart.FileHeader, error)
	RawPath() string
	Path() string
	LogPrintln(v ...any) *BasicInjector
	LogPrint(v ...any) *BasicInjector
	LogPrintf(format string, v ...any) *BasicInjector
	LogDebug(v ...any) *BasicInjector
	LogTrace(v ...any) *BasicInjector
	LogInfo(v ...any) *BasicInjector
	LogWarn(v ...any) *BasicInjector
	LogError(v ...any) *BasicInjector
	LogTracef(format string, v ...any) *BasicInjector
	LogDebugf(format string, v ...any) *BasicInjector
	LogInfof(format string, v ...any) *BasicInjector
	LogWarnf(format string, v ...any) *BasicInjector
	LogErrorf(format string, v ...any) *BasicInjector
	request() *http.Request
	response() http.ResponseWriter
	ServeStaticFile(filePath string, statusCode int) Result
	RequestHeader(key string) string
	HasRequestHeader(key string) bool
	LookupRequestHeader(key string) (string, bool)
	EqualIfExistRequestHeader(key, expected string) bool
	ContainsIfExistRequestHeader(key, value string) bool
	RealIp() string
	FormParams() (url.Values, error)
	MultipartForm() (*multipart.Form, error)
	Cookie(name string) (*http.Cookie, error)
	SetCookie(cookie *http.Cookie)
	Cookies() []*http.Cookie
	WrapStatusNotAcceptable(_error string, extValues ...any) Result
	LookupQueryParam(key string) bool
}

type BasicInjector struct {
	pathParameters    httprouter.Params
	r                 *http.Request
	w                 http.ResponseWriter
	requestBody       any
	extInjections     Map[string, any]
	defaultErrorCodes Map[int, string]
	rawPath           string
	logger            logger
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

func (s *BasicInjector) WrapStatusNotAcceptable(_error string, extValues ...any) Result {
	return s.WrapJsonErr(_error, s.defaultErrorCodes[http.StatusNotAcceptable], http.StatusNotAcceptable, extValues...)
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

func (s *BasicInjector) Path() string {
	return s.r.RequestURI
}

func (s *BasicInjector) RawPath() string {
	return s.rawPath
}

func (s *BasicInjector) LogPrintln(v ...any) *BasicInjector {
	s.logger.println(append([]any{"path=" + s.Path()}, v...)...)
	return s
}

func (s *BasicInjector) LogPrint(v ...any) *BasicInjector {
	s.logger.print(append([]any{"path=" + s.Path() + " "}, v...)...)
	return s
}

func (s *BasicInjector) LogPrintf(format string, v ...any) *BasicInjector {
	s.logger.printf("path="+s.Path()+" "+format, v...)
	return s
}

func (s *BasicInjector) LogTrace(v ...any) *BasicInjector {
	s.logger.println(append([]any{"[TRACE] path=" + s.Path()}, v...)...)
	return s
}

func (s *BasicInjector) LogDebug(v ...any) *BasicInjector {
	s.logger.println(append([]any{"[DEBUG] path=" + s.Path()}, v...)...)
	return s
}

func (s *BasicInjector) LogInfo(v ...any) *BasicInjector {
	s.logger.println(append([]any{"[INFO] path=" + s.Path()}, v...)...)
	return s
}

func (s *BasicInjector) LogWarn(v ...any) *BasicInjector {
	s.logger.println(append([]any{"[WARN] path=" + s.Path()}, v...)...)
	return s
}

func (s *BasicInjector) LogError(v ...any) *BasicInjector {
	s.logger.println(append([]any{"[ERROR] path=" + s.Path()}, v...)...)
	return s
}

func (s *BasicInjector) LogTracef(format string, v ...any) *BasicInjector {
	s.logger.printf("[TRACE] path="+s.Path()+" "+format, v...)
	return s
}

func (s *BasicInjector) LogDebugf(format string, v ...any) *BasicInjector {
	s.logger.printf("[DEBUG] path="+s.Path()+" "+format, v...)
	return s
}

func (s *BasicInjector) LogInfof(format string, v ...any) *BasicInjector {
	s.logger.printf("[INFO] path="+s.Path()+" "+format, v...)
	return s
}

func (s *BasicInjector) LogWarnf(format string, v ...any) *BasicInjector {
	s.logger.printf("[WARN] path="+s.Path()+" "+format, v...)
	return s
}

func (s *BasicInjector) LogErrorf(format string, v ...any) *BasicInjector {
	s.logger.printf("[ERROR] path="+s.Path()+" "+format, v...)
	return s
}

func (s *BasicInjector) response() http.ResponseWriter {
	return s.w
}

func (s *BasicInjector) request() *http.Request {
	return s.r
}

func (s *BasicInjector) ServeStaticFile(filePath string, statusCode int) Result {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return s.WrapInternalErr("while serving static file, err=" + err.Error())
	}

	return s.Wrap(file, statusCode)
}

func (s *BasicInjector) RequestHeader(key string) string {
	return s.RequestHeaders().Get(key)
}

func (s *BasicInjector) HasRequestHeader(key string) bool {
	return s.RequestHeaders().Get(key) != ""
}

func (s *BasicInjector) LookupRequestHeader(key string) (string, bool) {
	return s.RequestHeader(key), s.HasRequestHeader(key)
}

func (s *BasicInjector) EqualIfExistRequestHeader(key, expected string) bool {
	if !s.HasRequestHeader(key) {
		return true
	}

	return s.RequestHeader(key) == expected
}

func (s *BasicInjector) ContainsIfExistRequestHeader(key, value string) bool {
	if !s.HasRequestHeader(key) {
		return true
	}

	return strings.Contains(s.RequestHeader(key), value)
}

func (s *BasicInjector) RealIp() string {
	if ip := s.r.Header.Get("X-Forwarded-For"); ip != "" {
		i := strings.IndexAny(ip, ",")
		if i > 0 {
			xffip := strings.TrimSpace(ip[:i])
			xffip = strings.TrimPrefix(xffip, "[")
			xffip = strings.TrimSuffix(xffip, "]")
			return xffip
		}

		return ip
	}

	if ip := s.r.Header.Get("X-Real-Ip"); ip != "" {
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")

		return ip
	}

	ra, _, _ := net.SplitHostPort(s.r.RemoteAddr)

	return ra
}

func (s *BasicInjector) FormParams() (url.Values, error) {
	if strings.HasPrefix(s.r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := s.r.ParseMultipartForm(32 << 20); err != nil {
			return nil, err
		}
	} else {
		if err := s.r.ParseForm(); err != nil {
			return nil, err
		}
	}

	return s.r.Form, nil
}

func (s *BasicInjector) FormFile(name string) (*multipart.FileHeader, error) {
	f, fh, err := s.r.FormFile(name)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return fh, nil
}

func (s *BasicInjector) MultipartForm() (*multipart.Form, error) {
	err := s.r.ParseMultipartForm(32 << 20)

	return s.r.MultipartForm, err
}

func (s *BasicInjector) Cookie(name string) (*http.Cookie, error) {
	return s.r.Cookie(name)
}

func (s *BasicInjector) SetCookie(cookie *http.Cookie) {
	http.SetCookie(s.w, cookie)
}

func (s *BasicInjector) Cookies() []*http.Cookie {
	return s.r.Cookies()
}

func (s *BasicInjector) LookupQueryParam(key string) bool {
	return s.Query(key) != ""
}
