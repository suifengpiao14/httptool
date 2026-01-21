package httputil

import "io"

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
