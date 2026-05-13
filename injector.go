package flex

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
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
	"regexp"
	"strconv"
	"strings"
	"time"
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

func (s *BasicInjector) ResponseWriter() http.ResponseWriter {
	return s.w
}

func (s *BasicInjector) Request() *http.Request {
	return s.r
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
	http.ServeFile(s.w, s.r, filename)

	return s.Wrap(nil, statusCode)
}

func (s *BasicInjector) RequestBodyFFVM() []ffvm.ValidatorIssue {
	return ffvm.Validate(s.requestBody)
}

// ─── Typed Query Parameter Helpers ───────────────────────────────────────────

func (s *BasicInjector) QueryInt(key string) (int, error) {
	v := s.Query(key)
	if v == "" {
		return 0, fmt.Errorf("query param %q not found", key)
	}

	return strconv.Atoi(v)
}

func (s *BasicInjector) QueryInt64(key string) (int64, error) {
	v := s.Query(key)
	if v == "" {
		return 0, fmt.Errorf("query param %q not found", key)
	}
	return strconv.ParseInt(v, 10, 64)
}

func (s *BasicInjector) QueryFloat64(key string) (float64, error) {
	v := s.Query(key)
	if v == "" {
		return 0, fmt.Errorf("query param %q not found", key)
	}
	return strconv.ParseFloat(v, 64)
}

func (s *BasicInjector) QueryBool(key string) (bool, error) {
	v := s.Query(key)
	if v == "" {
		return false, fmt.Errorf("query param %q not found", key)
	}
	return strconv.ParseBool(v)
}

// QueryDefault returns the query param or a typed default — parsed via a
// provided parse function. Useful for optional numerics without boilerplate.
func QueryDefault[T any](s *BasicInjector, key string, parse func(string) (T, error), def T) T {
	v := s.Query(key)
	if v == "" {
		return def
	}
	parsed, err := parse(v)
	if err != nil {
		return def
	}
	return parsed
}

// QuerySlice splits a repeated comma-separated query param into a slice.
// e.g. ?tags=go,api,flex → ["go", "api", "flex"]
func (s *BasicInjector) QuerySlice(key string) []string {
	v := s.Query(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}

// QueryAll returns every value for a key (e.g. ?id=1&id=2).
func (s *BasicInjector) QueryAll(key string) []string {
	return s.URL().Query()[key]
}

// RequireQuery returns the value or immediately panics with a 400.
func (s *BasicInjector) RequireQuery(key string) string {
	v := s.Query(key)
	if v == "" {
		panic(s.WrapBadRequestErr(fmt.Sprintf("missing required query parameter: %q", key)))
	}
	return v
}

// ─── Typed Path Parameter Helpers ────────────────────────────────────────────

func (s *BasicInjector) PathParamInt(key string) (int, error) {
	return strconv.Atoi(s.PathParameter(key))
}

func (s *BasicInjector) PathParamInt64(key string) (int64, error) {
	return strconv.ParseInt(s.PathParameter(key), 10, 64)
}

// RequirePathParamInt returns the path param as int or panics with a 400.
func (s *BasicInjector) RequirePathParamInt(key string) int {
	v, err := s.PathParamInt(key)
	if err != nil {
		panic(s.WrapBadRequestErr(fmt.Sprintf("path parameter %q must be an integer", key)))
	}
	return v
}

// ─── Typed Request Body ───────────────────────────────────────────────────────

// BodyAs decodes the raw request body into a target type T without relying on
// the pre-wired bodyType — useful in middleware or when the body type is only
// known at call time.
func BodyAs[T any](s *BasicInjector) (T, error) {
	var target T
	if err := s.jsonHandler.NewDecoder(s.r.Body).Decode(&target); err != nil {
		return target, err
	}
	return target, nil
}

// MustBodyAs is like BodyAs but panics with a 400 on failure.
func MustBodyAs[T any](s *BasicInjector) T {
	v, err := BodyAs[T](s)
	if err != nil {
		panic(s.WrapBadRequestErr("could not decode request body: " + err.Error()))
	}
	return v
}

// ─── Response Helpers ─────────────────────────────────────────────────────────

func (s *BasicInjector) WrapCreated(response any) Result {
	return s.Wrap(response, http.StatusCreated)
}

func (s *BasicInjector) WrapAccepted(response any) Result {
	return s.Wrap(response, http.StatusAccepted)
}

func (s *BasicInjector) WrapUnauthorizedErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusUnauthorized], http.StatusUnauthorized)
}

func (s *BasicInjector) WrapConflictErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusConflict], http.StatusConflict)
}

func (s *BasicInjector) WrapUnprocessableErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusUnprocessableEntity], http.StatusUnprocessableEntity)
}

func (s *BasicInjector) WrapServiceUnavailableErr(err any) Result {
	return s.WrapJsonErr(err, s.defaultErrorCodes[http.StatusServiceUnavailable], http.StatusServiceUnavailable)
}

// WrapError inspects a standard error and maps known sentinel types to the
// appropriate HTTP response. Falls back to 500.
func (s *BasicInjector) WrapError(err error) Result {
	if err == nil {
		return s.WrapNoContent()
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return s.WrapJsonErr(httpErr.Message, s.defaultErrorCodes[httpErr.StatusCode], httpErr.StatusCode)
	}
	return s.WrapInternalErr(err.Error())
}

// HTTPError is a structured error carrying an HTTP status code.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Message)
}

func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Message: message}
}

// WrapXML serialises the response as XML instead of JSON.
func (s *BasicInjector) WrapXML(response any, statusCode int) Result {
	b, err := xml.Marshal(response)
	if err != nil {
		return s.WrapInternalErr("xml marshal error: " + err.Error())
	}
	s.SetContentType("application/xml; charset=utf-8")
	return s.Wrap(b, statusCode)
}

// ─── Redirect ─────────────────────────────────────────────────────────────────

func (s *BasicInjector) Redirect(url string, statusCode int) Result {
	http.Redirect(s.w, s.r, url, statusCode)
	return s.Wrap(nil, statusCode)
}

func (s *BasicInjector) RedirectPermanent(url string) Result {
	return s.Redirect(url, http.StatusMovedPermanently)
}

func (s *BasicInjector) RedirectTemporary(url string) Result {
	return s.Redirect(url, http.StatusTemporaryRedirect)
}

// ─── File / Download Responses ────────────────────────────────────────────────

// ServeFileDownload sends a file as an attachment with the given filename hint.
func (s *BasicInjector) ServeFileDownload(filePath, downloadName string) Result {
	s.SetResponseHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	return s.ServeStaticFile(filePath, http.StatusOK)
}

// WriteBytes writes raw bytes directly to the response writer and returns a
// sentinel Result so the framework skips its own serialisation step.
func (s *BasicInjector) WriteBytes(data []byte, contentType string, statusCode int) Result {
	s.SetContentType(contentType)
	s.w.WriteHeader(statusCode)
	_, _ = s.w.Write(data)
	return s.Wrap(nil, statusCode)
}

// ─── Server-Sent Events ───────────────────────────────────────────────────────

// SSEEvent writes a single SSE event to the response. The caller is responsible
// for setting "Content-Type: text/event-stream" before the first write.
func (s *BasicInjector) SSEEvent(event, data string) error {
	flusher, ok := s.w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported by this ResponseWriter")
	}
	if event != "" {
		fmt.Fprintf(s.w, "event: %s\n", event)
	}
	fmt.Fprintf(s.w, "data: %s\n\n", data)
	flusher.Flush()
	return nil
}

// SSEJsonEvent marshals v and emits it as an SSE event.
func (s *BasicInjector) SSEJsonEvent(event string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.SSEEvent(event, string(b))
}

// StartSSE sets the required SSE headers and returns whether flushing is
// supported. Call this once before any SSEEvent calls.
func (s *BasicInjector) StartSSE() bool {
	s.SetContentType("text/event-stream")
	s.SetResponseHeader("Cache-Control", "no-cache")
	s.SetResponseHeader("Connection", "keep-alive")
	s.SetResponseHeader("X-Accel-Buffering", "no")
	_, ok := s.w.(http.Flusher)
	return ok
}

// ─── Request ID / Tracing ─────────────────────────────────────────────────────

// RequestID returns the pre-assigned request ID for this injector.
func (s *BasicInjector) RequestID() string {
	return s.id
}

// TraceID returns the trace ID from common headers (B3, W3C traceparent, or
// falls back to the internal request ID).
func (s *BasicInjector) TraceID() string {
	if v := s.GetRequestHeader("X-B3-TraceId"); v != "" {
		return v
	}
	if v := s.GetRequestHeader("Traceparent"); v != "" {
		// traceparent: 00-<trace-id>-<parent-id>-<flags>
		parts := strings.Split(v, "-")
		if len(parts) == 4 {
			return parts[1]
		}
	}
	return s.id
}

// ─── Pagination ───────────────────────────────────────────────────────────────

type PageRequest struct {
	Page    int
	PerPage int
	Offset  int
}

// Pagination extracts page/per_page (or limit/offset) query params with
// sensible defaults and a configurable max page size.
func (s *BasicInjector) Pagination(defaultPerPage, maxPerPage int) PageRequest {
	page := QueryDefault(s, "page", strconv.Atoi, 1)
	if page < 1 {
		page = 1
	}
	perPage := QueryDefault(s, "per_page", strconv.Atoi, defaultPerPage)
	if perPage < 1 || perPage > maxPerPage {
		perPage = defaultPerPage
	}
	return PageRequest{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

// ─── Header Assertions ───────────────────────────────────────────────────────

// RequireContentType panics with a 415 if the request Content-Type does not
// contain the expected value (e.g. "application/json").
func (s *BasicInjector) RequireContentType(expected string) {
	ct := s.GetRequestHeader("Content-Type")
	if !strings.Contains(ct, expected) {
		panic(s.Wrap(M{
			"error": fmt.Sprintf("Content-Type must contain %q, got %q", expected, ct),
			"code":  s.defaultErrorCodes[http.StatusUnsupportedMediaType],
		}, http.StatusUnsupportedMediaType))
	}
}

// RequireAccepts panics with a 406 if the Accept header does not match.
func (s *BasicInjector) RequireAccepts(contentType string) {
	accept := s.GetRequestHeader("Accept")
	if accept != "" && accept != "*/*" && !strings.Contains(accept, contentType) {
		panic(s.WrapStatusNotAcceptable(fmt.Sprintf("client does not accept %q", contentType)))
	}
}

// ─── Bearer Token Extraction ──────────────────────────────────────────────────

// BearerToken extracts the token from "Authorization: Bearer <token>".
// Returns ("", false) if the header is absent or malformed.
func (s *BasicInjector) BearerToken() (string, bool) {
	auth := s.GetRequestHeader("Authorization")
	if auth == "" {
		return "", false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}

// RequireBearerToken returns the bearer token or panics with a 401.
func (s *BasicInjector) RequireBearerToken() string {
	token, ok := s.BearerToken()
	if !ok {
		panic(s.WrapUnauthorizedErr("missing or malformed Authorization header"))
	}
	return token
}

// ─── Cookie Helpers ───────────────────────────────────────────────────────────

// SetSecureCookie sets a cookie with secure, HttpOnly and SameSite=Strict
// defaults — a safe baseline for session tokens.
func (s *BasicInjector) SetSecureCookie(name, value string, maxAge int) {
	s.SetCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

// DeleteCookie expires a cookie immediately.
func (s *BasicInjector) DeleteCookie(name string) {
	s.SetCookie(&http.Cookie{
		Name:    name,
		Value:   "",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
		Path:    "/",
	})
}

// CookieValue returns the cookie value or a default string.
func (s *BasicInjector) CookieValue(name, defaultValue string) string {
	c, err := s.Cookie(name)
	if err != nil {
		return defaultValue
	}
	return c.Value
}

// ─── Value Store Typed Helpers ────────────────────────────────────────────────

// SetTypedValue stores a typed value without boxing it in an interface at the
// call site (generic convenience wrapper around SetValue).
func SetTypedValue[T any](s *BasicInjector, key string, value T) {
	s.SetValue(key, value)
}

// TypedValue retrieves a value from the store and type-asserts it to T.
// Returns the zero value and false if the key is absent or the type mismatches.
func TypedValue[T any](s *BasicInjector, key string) (T, bool) {
	v, ok := s.LookupValue(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := v.(T)
	return typed, ok
}

// MustTypedValue retrieves a value or panics with a 500 if absent/wrong type.
func MustTypedValue[T any](s *BasicInjector, key string) T {
	v, ok := TypedValue[T](s, key)
	if !ok {
		panic(s.WrapInternalErr(fmt.Sprintf("injector value %q not found or wrong type", key)))
	}
	return v
}

// ─── Structured Logging Extras ────────────────────────────────────────────────

// LogWith returns a child injector with the extra key=value appended to every
// subsequent log call by embedding the pair in extInjections as a log-prefix.
func (s *BasicInjector) LogFields(kv ...any) *BasicInjector {
	if len(kv)%2 != 0 {
		kv = append(kv, "MISSING")
	}
	pairs := make([]string, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		pairs = append(pairs, fmt.Sprintf("%v=%v", kv[i], kv[i+1]))
	}
	s.logger.println(append([]any{"[FIELDS] path=" + s.Path()}, strings.Join(pairs, " "))...)
	return s
}

// ─── Request Introspection ────────────────────────────────────────────────────

// IsJSON reports whether the request Content-Type is application/json.
func (s *BasicInjector) IsJSON() bool {
	return strings.Contains(s.GetRequestHeader("Content-Type"), "application/json")
}

// IsMultipart reports whether the request is a multipart/form-data upload.
func (s *BasicInjector) IsMultipart() bool {
	return strings.HasPrefix(s.GetRequestHeader("Content-Type"), "multipart/form-data")
}

// IsWebSocket reports whether the request is a WebSocket upgrade.
func (s *BasicInjector) IsWebSocket() bool {
	return strings.EqualFold(s.GetRequestHeader("Upgrade"), "websocket")
}

// AcceptsJSON reports whether the client accepts application/json.
func (s *BasicInjector) AcceptsJSON() bool {
	accept := s.GetRequestHeader("Accept")
	return accept == "" || strings.Contains(accept, "application/json") || strings.Contains(accept, "*/*")
}

// IsTLS reports whether the underlying connection used TLS.
func (s *BasicInjector) IsTLS() bool {
	return s.r.TLS != nil
}

// IsHTMX reports whether the request came from an HTMX-driven client.
func (s *BasicInjector) IsHTMX() bool {
	return s.GetRequestHeader("HX-Request") == "true"
}

// UserAgent returns the request User-Agent header.
func (s *BasicInjector) UserAgent() string {
	return s.GetRequestHeader("User-Agent")
}

// Referer returns the HTTP Referer header.
func (s *BasicInjector) Referer() string {
	return s.r.Referer()
}

// ─── Input Validation Helpers ─────────────────────────────────────────────────

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail returns false if the string is not a plausible email address.
func ValidateEmail(email string) bool {
	return emailRegexp.MatchString(email)
}

// AssertQuery panics with a 400 if the query param fails the provided predicate.
func (s *BasicInjector) AssertQuery(key string, predicate func(string) bool, errMsg string) string {
	v := s.RequireQuery(key)
	if !predicate(v) {
		panic(s.WrapBadRequestErr(errMsg))
	}
	return v
}

// ─── Timing / Deadline Helpers ────────────────────────────────────────────────

// WithTimeout derives a context with a timeout and replaces the injector's
// context. The returned cancel func must be deferred by the caller.
func (s *BasicInjector) WithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(s.Context(), d)
	s.SetContext(ctx)
	return ctx, cancel
}

// WithDeadline derives a context with a deadline and replaces the injector's
// context. The returned cancel func must be deferred by the caller.
func (s *BasicInjector) WithDeadline(t time.Time) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(s.Context(), t)
	s.SetContext(ctx)
	return ctx, cancel
}
