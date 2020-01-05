package minonsdk

import "io"

type progressReader struct {
	io.Reader
	Reporter func(r int)
}

func (r *progressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Reporter(n)
	return
}
