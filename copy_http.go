package httptool

import (
	"bytes"
	"io"
	"net/http"
)

// deepCopyHeader 深拷贝 http.Header
func deepCopyHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	copyHeader := make(http.Header, len(h))
	for k, vv := range h {
		dst := make([]string, len(vv))
		copy(dst, vv)
		copyHeader[k] = dst
	}
	return copyHeader
}

// CopyRequest 深拷贝 http.Request，Body 可重复读取
func CopyRequest(r *http.Request, body []byte) (copyRequest *http.Request, reqBody []byte, err error) {
	if r == nil {
		return nil, nil, nil
	}
	// 基于原始 request 克隆
	reqCopy := r.Clone(r.Context())

	reqCopy.Header = r.Header.Clone()
	reqCopy.Trailer = deepCopyHeader(r.Trailer)
	if len(body) > 0 {
		reqCopy.Body = io.NopCloser(bytes.NewBuffer(body))
		if r.GetBody == nil {
			reqCopy.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(body)), nil }
		}
		reqCopy.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(body)), nil }
		return reqCopy, body, nil
	}

	if r.GetBody != nil { // 如果有 GetBody，则优先使用 GetBody 获取 body 提升性能
		bodyReader, err := r.GetBody()
		if err != nil {
			return reqCopy, reqBody, err
		}
		defer bodyReader.Close()
		reqBody, err = io.ReadAll(bodyReader) // 读取原始 request body
		if err != nil {
			return reqCopy, reqBody, err // CopyResponse 时会忽略err，但是需要reqCopy ，所以这里同步返回reqCopy
		}
		// 复制用 body
		reqCopy.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		reqCopy.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(reqBody)), nil }
	}

	if r.Body != nil {
		reqBody, err = io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return reqCopy, reqBody, err // CopyResponse 时会忽略err，但是需要reqCopy ，所以这里同步返回reqCopy
		}

		// 恢复原始 request
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(reqBody)), nil }

		// 复制用 body
		reqCopy.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		reqCopy.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(reqBody)), nil }
	}

	return reqCopy, reqBody, nil
}

// CopyResponse 深拷贝 http.Response，Body 可重复读取
func CopyResponse(resp *http.Response, body []byte) (copyResponse *http.Response, rspBody []byte, err error) {
	rspBody = body
	if resp == nil {
		return nil, nil, nil
	}
	respCopy := *resp // 浅拷贝结构体
	respCopy.Header = deepCopyHeader(resp.Header)
	respCopy.Trailer = deepCopyHeader(resp.Trailer)
	respCopy.Request, _, _ = CopyRequest(resp.Request, nil) // request 可能已经被读取，所以需要忽略错误
	// if err != nil {
	// 	return nil, err
	// }
	if body != nil {
		respCopy.Body = io.NopCloser(bytes.NewBuffer(body))
		return &respCopy, body, nil // 如果有 body，则直接返回
	}

	if resp.Body != nil {
		rspBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, rspBody, err
		}
		// 恢复原始 response
		resp.Body = io.NopCloser(bytes.NewBuffer(rspBody))
		// 复制用 body
		respCopy.Body = io.NopCloser(bytes.NewBuffer(rspBody))
	}

	return &respCopy, rspBody, nil
}
