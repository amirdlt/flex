package flex

import (
	"context"
	"fmt"
	"github.com/amirdlt/ffvm"
	. "github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
)

type NoBody struct{}

var noBody NoBody

type Injector interface {
	ResponseHeaders() http.Header
	URL() *url.URL
	GetRequestHeader(key string) string
	Host() string
	Method() string
	ContentLength() int64
	WrapOk(response any) Result
	WrapNoContent() Result
	WrapTooManyRequestsErr(err any) Result
	SetContentType(contentType string)
	RemoteAddr() string
	Path() string
	request() *http.Request
	response() http.ResponseWriter
	ServeStaticFile(filePath string, statusCode int) Result
	RealIp() string
}

type BasicInjector struct {
	pathParameters    httprouter.Params
	r                 *http.Request
	w                 http.ResponseWriter
	requestBody       any
	bodyProcessed     bool
	extInjections     Map[string, any]
	defaultErrorCodes Map[int, string]
	rawPath           string
	logger            logger
	ctx               context.Context
	id                string
	jsonHandler       JsonHandler
	bodyType          reflect.Type
}

func (s *BasicInjector) PathParameter(key string) string {
	return s.pathParameters.ByName(key)
}

func (s *BasicInjector) RequestBody() any {
	s.readBody()
	return s.requestBody
}

func (s *BasicInjector) readBody() {
	if s.bodyProcessed {
		return
	}

	defer func() {
		_ = s.r.Body.Close()
		s.bodyProcessed = true
	}()

	requestBodyPtr := reflect.New(s.bodyType)
	val := reflect.ValueOf(requestBodyPtr.Elem().Interface())
	kind := val.Kind()
	if (kind == reflect.Array || kind == reflect.Slice) &&
		val.Type().Elem().Kind() == reflect.Uint8 || kind == reflect.String {
		arr, err := io.ReadAll(s.r.Body)
		if err != nil {
			panic(s.WrapBadRequestErr("could not read body, err=" + err.Error()))
		}

		if kind == reflect.String {
			s.requestBody = string(arr)
		} else {
			s.requestBody = arr
		}

		return
	}

	if reflect.TypeOf(noBody) != s.bodyType {
		if err := s.jsonHandler.NewDecoder(s.r.Body).Decode(requestBodyPtr.Interface()); err != nil {
			panic(s.WrapBadRequestErr("could not read body as a valid json, err=" + err.Error()))
		}

		s.requestBody = requestBodyPtr.Elem().Interface()
	}
}

func (s *BasicInjector) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}

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

func (s *BasicInjector) Wrap(response any, statusCode int) Result {
	return Result{
		responseBody: response,
		statusCode:   statusCode,
	}
}

func (s *BasicInjector) WrapWithContentType(response any, statusCode int, contentType string) Result {
	s.SetResponseHeader("Content-Type", contentType)
	return Result{
		responseBody: response,
		statusCode:   statusCode,
	}
}

func (s *BasicInjector) WrapOk(response any) Result {
	return Result{
		responseBody: response,
	}
}

func (s *BasicInjector) WrapNoContent() Result {
	return s.Wrap(nil, http.StatusNoContent)
}

func (s *BasicInjector) WrapJsonErr(err any, code string, statusCode int) Result {
	return s.WrapWithContentType(M{
		"error": err,
		"code":  code,
	}, statusCode, "application/json")
}

func (s *BasicInjector) WrapInvalidBody(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest)
}

func (s *BasicInjector) WrapNotFoundErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusNotFound], http.StatusNotFound)
}

func (s *BasicInjector) WrapForbiddenErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusForbidden], http.StatusForbidden)
}

func (s *BasicInjector) WrapInternalErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusInternalServerError], http.StatusInternalServerError)
}

func (s *BasicInjector) WrapBadRequestErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusBadRequest], http.StatusBadRequest)
}

func (s *BasicInjector) WrapStatusNotAcceptable(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusNotAcceptable], http.StatusNotAcceptable)
}

func (s *BasicInjector) WrapTooManyRequestsErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusTooManyRequests], http.StatusTooManyRequests)
}

func (s *BasicInjector) WrapTextPlain(response any, statusCode int) Result {
	return s.WrapWithContentType(fmt.Sprint(response), statusCode, "text/plain")
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
	return strings.TrimSuffix(s.r.URL.Path, "/")
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

func (s *BasicInjector) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *BasicInjector) DefaultServeFile(filename string, statusCode int) Result {
	http.ServeFile(s.response(), s.request(), filename)
	return s.Wrap(nil, statusCode)
}

func (s *BasicInjector) RequestBodyFFVM() []ffvm.ValidatorIssue {
	return ffvm.Validate(s.requestBody)
}
