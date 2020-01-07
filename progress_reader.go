package sdk

import "io"

type ProgressReader struct {
	io.Reader
	Reporter func(r int)
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Reporter(n)
	return
}
