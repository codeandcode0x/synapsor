package util

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	netUrl "net/url"
	"strings"
	"time"
)

// HttpMethod http method
type HttpMethod string

const (
	MethodGet     HttpMethod = http.MethodGet
	MethodHead    HttpMethod = http.MethodHead
	MethodPost    HttpMethod = http.MethodPost
	MethodPut     HttpMethod = http.MethodPut
	MethodPatch   HttpMethod = http.MethodPatch // RFC 5789
	MethodDelete  HttpMethod = http.MethodDelete
	MethodConnect HttpMethod = http.MethodConnect
	MethodOptions HttpMethod = http.MethodOptions
	MethodTrace   HttpMethod = http.MethodTrace
)

// MimeType mime type
type MimeType string

const (
	MimeTextHtml                      MimeType = "text/html"
	MimeTextPlain                     MimeType = "text/plain"
	MimeTextXml                       MimeType = "text/xml"
	MimeApplicationJson               MimeType = "application/json"
	MimeApplicationXWwwFormUrlencoded MimeType = "application/x-www-form-urlencoded"
	MimeMultipartFormData             MimeType = "multipart/form-data"
)

// HttpClient
type HttpClient struct {
	client http.Client
}

// NewHttpClient 创建新的http client
// 创建client应该复用而不是每次创建，创建的client是协程安全的（详见http.Client注释说明）
func NewHttpClient() *HttpClient {
	return new(HttpClient)
}

// WithClientTransport client transport
func (c *HttpClient) WithTransport(transport http.RoundTripper) *HttpClient {
	c.client.Transport = transport
	return c
}

// WithCheckRedirect client checkRedirect
func (c *HttpClient) WithCheckRedirect(checkRedirect func(req *http.Request, via []*http.Request) error) *HttpClient {
	c.client.CheckRedirect = checkRedirect
	return c
}

// WithJar client jar
func (c *HttpClient) WithJar(jar http.CookieJar) *HttpClient {
	c.client.Jar = jar
	return c
}

// WithTimeOut client timeout
func (c *HttpClient) WithTimeOut(timeout time.Duration) *HttpClient {
	c.client.Timeout = timeout
	return c
}

// Do 发http请求
func (c *HttpClient) Do(
	method HttpMethod, 
	url string, 
	body io.Reader, 
	options ...OptionFn ) (*http.Response, []byte, error) {
	// 设置参数
	opt := &Option{
		headers:     make(map[string]string),
		queryParams: make(map[string]string),
	}
	for _, option := range options {
		option(opt)
	}

	// url加入query参数
	if len(opt.queryParams) > 0 {
		query := (&netUrl.URL{}).Query()
		for key, value := range opt.queryParams {
			query.Add(key, value)
		}

		if strings.Contains(url, "?") {
			url += "&" + query.Encode()
		} else {
			url += "?" + query.Encode()
		}
	}

	log.Println("http request url:", url)

	// 构建request
	request, err := http.NewRequest(string(method), url, body)
	if err != nil {
		return nil, nil, err
	}

	// 设置header
	for key, value := range opt.headers {
		request.Header.Add(key, value)
	}

	// 发送请求
	response, err := c.client.Do(request)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return nil, nil, err
	}

	// 返回数据
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}

	return response, responseBody, nil
}

// http请求参数
type Option struct {
	// headers
	headers map[string]string

	// query参数
	queryParams map[string]string
}

// OptionFn
type OptionFn func(*Option)

// WithHeaders headers
func WithHeaders(headers map[string]string) OptionFn {
	return func(o *Option) {
		for key, value := range headers {
			o.headers[key] = value
		}
	}
}

// WithHeader add headers one by one
func WithHeader(key string, value string) OptionFn {
	return func(o *Option) {
		o.headers[key] = value
	}
}

// WithQueryParams queryParams
func WithQueryParams(queryParams map[string]string) OptionFn {
	return func(o *Option) {
		for key, value := range queryParams {
			o.queryParams[key] = value
		}
	}
}

// WithQueryParam add queryParams one by one
func WithQueryParam(key string, value string) OptionFn {
	return func(o *Option) {
		o.queryParams[key] = value
	}
}