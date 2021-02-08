package goz

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jarcoal/httpmock"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Request object
type Request struct {
	opts Options
	cli  *http.Client
	req  *http.Request
	body io.Reader
}

// Get send get request
func (r *Request) Get(uri string, opts ...Options) (*Response, error) {
	return r.Request("GET", uri, opts...)
}

// Post send post request
func (r *Request) Post(uri string, opts ...Options) (*Response, error) {
	return r.Request("POST", uri, opts...)
}

// Put send put request
func (r *Request) Put(uri string, opts ...Options) (*Response, error) {
	return r.Request("PUT", uri, opts...)
}

// Patch send patch request
func (r *Request) Patch(uri string, opts ...Options) (*Response, error) {
	return r.Request("PATCH", uri, opts...)
}

// Delete send delete request
func (r *Request) Delete(uri string, opts ...Options) (*Response, error) {
	return r.Request("DELETE", uri, opts...)
}

// Options send options request
func (r *Request) Options(uri string, opts ...Options) (*Response, error) {
	return r.Request("OPTIONS", uri, opts...)
}

// Request send request
func (r *Request) Request(method, uri string, opts ...Options) (*Response, error) {
	if len(opts) > 0 {
		r.opts = opts[0]
	}

	switch method {
	case http.MethodGet, http.MethodDelete:
		req, err := http.NewRequest(method, uri, nil)
		if err != nil {
			return nil, err
		}

		r.req = req
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodOptions:
		// parse body
		r.parseBody()

		req, err := http.NewRequest(method, uri, r.body)
		if err != nil {
			return nil, err
		}

		r.req = req
	default:
		return nil, errors.New("invalid request method")
	}

	// parseOptions
	r.parseOptions()

	// parseClient
	r.parseClient()

	// parse query
	r.parseQuery()

	// parse headers
	r.parseHeaders()

	// parse cookies
	r.parseCookies()

	if r.opts.Debug {
		// print request object
		dump, err := httputil.DumpRequest(r.req, true)
		if err == nil {
			log.Printf("\n%s", dump)
		}
	}

	_resp, err := r.cli.Do(r.req)

	resp := &Response{
		resp: _resp,
		req:  r.req,
		err:  err,
	}

	if err != nil {
		if r.opts.Debug {
			// print response err
			fmt.Println(err)
		}

		return resp, err
	}

	if r.opts.Debug {
		// print response data
		body, _ := resp.GetBody()
		fmt.Println(string(body))
	}

	return resp, nil
}

func (r *Request) parseOptions() {
	// default timeout 30s
	if r.opts.Timeout == 0 {
		r.opts.Timeout = 30
	}
	r.opts.timeout = time.Duration(r.opts.Timeout*1000) * time.Millisecond
}

func (r *Request) parseClient() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if r.opts.Proxy != "" {
		proxy, err := url.Parse(r.opts.Proxy)
		if err == nil {
			tr.Proxy = http.ProxyURL(proxy)
		}
	}

	if len(httpmock.GetCallCountInfo()) == 0 {
		r.cli = &http.Client{
			Timeout:   r.opts.timeout,
			Transport: tr,
		}
	}

	if len(httpmock.GetCallCountInfo()) > 0 {
		isInMock := false

		for k := range httpmock.GetCallCountInfo() {
			// 先用簡單的方式判斷，只要 API 的 Host + Path 相加的路徑符合，就對 API 進行隔離，雖然不是很精準的作法，但對我來說目前夠用
			if strings.Contains(k, r.req.URL.Host + r.req.URL.Path) {
				isInMock = true
			}
		}

		if isInMock {
			r.cli = &http.Client{
				Timeout: r.opts.timeout,
				// Transport: tr,
			}
		}

		if !isInMock {
			r.cli = &http.Client{
				Timeout: r.opts.timeout,
				Transport: tr,
			}
		}
	}
}

func (r *Request) parseQuery() {
	switch r.opts.Query.(type) {
	case string:
		str := r.opts.Query.(string)
		r.req.URL.RawQuery = str
	case map[string]interface{}:
		q := r.req.URL.Query()
		for k, v := range r.opts.Query.(map[string]interface{}) {
			if vv, ok := v.(string); ok {
				q.Set(k, vv)
				continue
			}
			if vv, ok := v.([]string); ok {
				for _, vvv := range vv {
					q.Add(k, vvv)
				}
			}
		}
		r.req.URL.RawQuery = q.Encode()
	}
}

func (r *Request) parseCookies() {
	switch r.opts.Cookies.(type) {
	case string:
		cookies := r.opts.Cookies.(string)
		r.req.Header.Add("Cookie", cookies)
	case map[string]string:
		cookies := r.opts.Cookies.(map[string]string)
		for k, v := range cookies {
			r.req.AddCookie(&http.Cookie{
				Name:  k,
				Value: v,
			})
		}
	case []*http.Cookie:
		cookies := r.opts.Cookies.([]*http.Cookie)
		for _, cookie := range cookies {
			r.req.AddCookie(cookie)
		}
	}
}

func (r *Request) parseHeaders() {
	if r.opts.Headers != nil {
		for k, v := range r.opts.Headers {
			if vv, ok := v.(string); ok {
				r.req.Header.Set(k, vv)
				continue
			}
			if vv, ok := v.([]string); ok {
				for _, vvv := range vv {
					r.req.Header.Add(k, vvv)
				}
			}
		}
	}
}

func (r *Request) parseBody() {
	// application/x-www-form-urlencoded
	if r.opts.FormParams != nil {
		values := url.Values{}
		for k, v := range r.opts.FormParams {
			if vv, ok := v.(string); ok {
				values.Set(k, vv)
			}
			if vv, ok := v.([]string); ok {
				for _, vvv := range vv {
					values.Add(k, vvv)
				}
			}
		}
		r.body = strings.NewReader(values.Encode())

		return
	}

	// application/json
	if r.opts.JSON != nil {
		b, err := json.Marshal(r.opts.JSON)
		if err == nil {
			r.body = bytes.NewReader(b)

			return
		}
	}

	return
}
