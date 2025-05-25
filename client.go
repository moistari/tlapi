package tlapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is a TL client.
type Client struct {
	cl        *http.Client
	Jar       []*http.Cookie
	Transport http.RoundTripper
	UserAgent string
}

// New creates a TL client.
func New(opts ...Option) *Client {
	cl := &Client{}
	for _, o := range opts {
		o(cl)
	}
	if cl.cl == nil {
		cl.cl = &http.Client{
			Transport: cl.Transport,
		}
	}
	return cl
}

// buildReq builds a request for the client.
func (cl *Client) buildReq(req *http.Request) error {
	switch {
	case len(cl.Jar) == 0:
		return errors.New("must supply cookie jar")
	case cl.UserAgent == "":
		return errors.New("must supply user-agent")
	}
	req.Header.Set("Cookie", cookies(cl.Jar))
	req.Header.Set("User-Agent", cl.UserAgent)
	return nil
}

// Do executes a request.
func (cl *Client) Do(ctx context.Context, req *http.Request, result any) error {
	if err := cl.buildReq(req); err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := cl.cl.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid http status %d", res.StatusCode)
	}
	dec := json.NewDecoder(res.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(result)
}

// Search searches for a query.
func (cl *Client) Search(ctx context.Context, query ...string) (*SearchResponse, error) {
	return Search(query...).Do(ctx, cl)
}

// Torrent retrieves a torrent for the id.
func (cl *Client) Torrent(ctx context.Context, id int) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.torrentleech.org/download/%d/%s", id, "a"), nil)
	if err != nil {
		return nil, err
	}
	if err := cl.buildReq(req); err != nil {
		return nil, err
	}
	res, err := cl.cl.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid http status %d", res.StatusCode)
	}
	return io.ReadAll(res.Body)
}

// Option is a TL client option.
type Option func(cl *Client)

// WithJar is an option to set the cookie jar used by the TL client.
func WithJar(jar []*http.Cookie) Option {
	return func(cl *Client) {
		cl.Jar = jar
	}
}

// WithTransport is a TL client option to set the http transport used by the TL
// client.
func WithTransport(transport http.RoundTripper) Option {
	return func(cl *Client) {
		cl.Transport = transport
	}
}

// WithCreds is a TL client option to set the cf_clearance, tluid, and tlpass
// cookies used by the TL client.
func WithCreds(cfClearance []string, uid, pass string) Option {
	return func(cl *Client) {
		cl.Jar = BuildJar(cfClearance, uid, pass)
	}
}

// WithUserAgent is a TL client option to set the user agent used by the TL
// client.
func WithUserAgent(userAgent string) Option {
	return func(cl *Client) {
		cl.UserAgent = userAgent
	}
}

// BuildJar creates a jar.
func BuildJar(cfClearance []string, uid, pass string) []*http.Cookie {
	return []*http.Cookie{
		{
			Domain: "torrentleech.org",
			Path:   "/",
			Name:   "cf_clearance",
			Value:  cfClearance[1],
			Secure: true,
		},
		{
			Domain: "torrentleech.org",
			Path:   "/",
			Name:   "cf_clearance",
			Value:  cfClearance[0],
			Secure: true,
		},
		{
			Domain:   "torrentleech.org",
			Path:     "/",
			Name:     "tluid",
			Value:    uid,
			HttpOnly: true,
			Secure:   true,
		},
		{
			Domain: "torrentleech.org",
			Path:   "/",
			Name:   "tlpass",
			Value:  pass,
			Secure: true,
		},
	}
}

// cookies returns a cookie string.
func cookies(cookies []*http.Cookie) string {
	var sb strings.Builder
	for i, cookie := range cookies {
		if i != 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(cookie.Name + "=" + cookie.Value)
	}
	return sb.String()
}
