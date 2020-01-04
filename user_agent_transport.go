package pkg

import "net/http"

type userAgentTransport struct {
	userAgent string
	rt        http.RoundTripper
}

func newUserAgentTransport(userAgent string, rt http.RoundTripper) *userAgentTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &userAgentTransport{userAgent: userAgent, rt: rt}
}

func (u *userAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Header.Get("User-Agent") == "" {
		r.Header.Set("User-Agent", u.userAgent)
	}

	return u.rt.RoundTrip(r)
}
