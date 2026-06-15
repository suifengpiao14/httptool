package httptool

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
)

// newBodyReader 创建一个轻量的、只读的 body reader。
// 使用 bytes.NewReader 而非 bytes.NewBuffer：
//  1. NewReader 更轻量（无写入相关字段）；
//  2. NewReader 不可变，不会被误写，语义更安全；
//  3. NewReader 支持多次 Seek 复用。
func newBodyReader(b []byte) io.ReadCloser {
	if len(b) == 0 {
		return http.NoBody
	}
	return io.NopCloser(bytes.NewReader(b))
}

// makeGetBody 构造 GetBody 闭包，统一使用 bytes.NewReader
func makeGetBody(b []byte) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		return newBodyReader(b), nil
	}
}

// CopyRequest 深拷贝 http.Request，Body 可重复读取。
//
// 优化点：
//  1. 不再重复 Clone Header（r.Clone 内部已深拷贝 Header/Trailer）；
//  2. GetBody 分支与 Body 分支互斥，避免重复读取 body；
//  3. 统一使用 bytes.NewReader 替代 bytes.NewBuffer，更轻量更安全；
//  4. 移除 GetBody 闭包的重复赋值。
func CopyRequest(r *http.Request, body []byte) (copyRequest *http.Request, reqBody []byte, err error) {
	reqBody = body
	if r == nil {
		return nil, nil, nil
	}

	// 基于原始 request 克隆（Clone 内部已深拷贝 Header、Trailer）
	reqCopy := r.Clone(r.Context())

	// 如果外部已传入 body，直接使用，无需读取
	if len(body) > 0 {
		reqCopy.Body = newBodyReader(body)
		reqCopy.GetBody = makeGetBody(body)
		return reqCopy, body, nil
	}

	// 优先使用 GetBody 获取 body（可重复获取，无需消费原始 Body）
	if r.GetBody != nil {
		bodyReader, err := r.GetBody()
		if err != nil {
			return reqCopy, reqBody, err
		}
		defer bodyReader.Close()
		reqBody, err = io.ReadAll(bodyReader)
		if err != nil {
			return reqCopy, reqBody, err
		}
		reqCopy.Body = newBodyReader(reqBody)
		reqCopy.GetBody = makeGetBody(reqBody)
		return reqCopy, reqBody, nil
	}

	// 没有 GetBody，则从 r.Body 读取（与 GetBody 分支互斥，避免重复读取）
	if r.Body != nil {
		if r.ContentLength == 0 {
			// 内容为空，直接返回空 body，不读取原始 request body
			reqCopy.Body = newBodyReader(nil)
		} else {
			reqBody, err = io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				return reqCopy, reqBody, err
			}
			// 恢复原始 request，使其可重复读取
			r.Body = newBodyReader(reqBody)
			r.GetBody = makeGetBody(reqBody)
			// 复制用 body
			reqCopy.Body = newBodyReader(reqBody)
			reqCopy.GetBody = makeGetBody(reqBody)
		}
	}

	return reqCopy, reqBody, nil
}

// CopyResponse 深拷贝 http.Response，Body 可重复读取。
//
// 优化点：统一使用 bytes.NewReader 替代 bytes.NewBuffer。
func CopyResponse(resp *http.Response, body []byte) (copyResponse *http.Response, rspBody []byte, err error) {
	rspBody = body
	if resp == nil {
		return nil, nil, nil
	}
	respCopy := *resp // 浅拷贝结构体
	respCopy.Header = resp.Header.Clone()
	respCopy.Trailer = resp.Trailer.Clone()
	if resp.Request != nil {
		respCopy.Request = resp.Request // 已有 request 引用，不重复复制 body
	} else {
		respCopy.Request, _, _ = CopyRequest(resp.Request, nil)
	}

	if body != nil {
		respCopy.Body = newBodyReader(body)
		return &respCopy, body, nil
	}

	if resp.Body != nil {
		if resp.ContentLength == 0 {
			// 内容为空，直接返回空 body，不读取原始 response body
			// （在 github.com/elazarl/goproxy 中会根据 body 对象是否变化删除 Content-length，
			//   导致 HEAD 无 body 请求无法中断）
			respCopy.Body = newBodyReader(nil)
		} else {
			rspBody, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, rspBody, err
			}
			// 恢复原始 response
			resp.Body = newBodyReader(rspBody)
			// 复制用 body
			respCopy.Body = newBodyReader(rspBody)
		}
	}

	return &respCopy, rspBody, nil
}

func DumpRequest(req *http.Request) string {
	reqRaw, err := httputil.DumpRequest(req, true) // 服务端
	if err != nil {
		reqRaw, err = httputil.DumpRequestOut(req, true) // 客户端
		if err != nil {
			reqRaw = []byte(err.Error())
		}
	}
	return string(reqRaw)
}

func DumpResponse(rsp *http.Response) string {
	rspRaw, err := httputil.DumpResponse(rsp, true)
	if err != nil {
		rspRaw = []byte(err.Error())
	}
	return string(rspRaw)
}
