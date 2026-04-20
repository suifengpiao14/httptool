package httptool

import (
	"bytes"
	"io"
	"net/http"
)

func ReadAll(body io.Reader) (b []byte, err error) {
	if body == nil {
		return nil, nil
	}
	if readerCloser, ok := body.(io.ReadCloser); ok {
		defer readerCloser.Close()
	}
	b, err = io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DrainBody copy from net/http/httputil/dump.go
func DrainBody(b io.ReadCloser) (r1 *bytes.Buffer, r2 io.ReadCloser, err error) {
	var buf bytes.Buffer
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return &buf, http.NoBody, nil
	}

	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return &buf, io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
