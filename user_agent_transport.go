package minonsdk

import "net/http"

type minionTransport struct {
	userAgent string
	rt        http.RoundTripper
}

func NewUserAgentTransport(userAgent string, rt http.RoundTripper) *minionTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &minionTransport{userAgent: userAgent, rt: rt}
}

func (u *minionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Header.Get("User-Agent") == "" {
		r.Header.Set("User-Agent", u.userAgent)
	}
	r.Header.Set("Content-Type", "application/json")

	return u.rt.RoundTrip(r)
}
