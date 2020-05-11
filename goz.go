package goz

// NewClient new request object
func NewClient(opts ...Options) *Request {
	req := &Request{}

	if len(opts) > 0 {
		req.opts = mergeHeaders(defaultHeader(), opts[0])
	}

	return req
}

// 合并用户提供的header头字段信息，用户提供的header头优先于默认头字段信息
func mergeHeaders(default_headers Options, headers ...Options) Options {
	if len(headers) == 0 {
		default_headers = defaultHeader()
	} else {
		for key, value := range default_headers.Headers {
			if _, ok := headers[0].Headers[key]; !ok {
				headers[0].Headers[key] = value
			}
		}
	}
	return headers[0]
}

func defaultHeader() Options {
	headers := Options{
		Headers: map[string]interface{}{
			"User-Agent":                "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.81 Safari/537.36 SE 2.X MetaSr 1.0",
			"Accept":                    "text/html,application/json,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
			"Accept-Encoding":           "gzip, deflate",
			"Accept-Language":           "zh-CN,zh;q=0.9",
			"Upgrade-Insecure-Requests": "1",
			"Cache-Control":             "max-age=0",
		},
	}
	return headers
}
