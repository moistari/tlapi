package tlapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client is a TL client.
type Client struct {
	cl        *http.Client
	Jar       http.CookieJar
	Transport http.RoundTripper
}

// New creates a TL client.
func New(opts ...Option) *Client {
	cl := &Client{}
	for _, o := range opts {
		o(cl)
	}
	if cl.cl == nil {
		cl.cl = &http.Client{
			Jar:       cl.Jar,
			Transport: cl.Transport,
		}
	}
	return cl
}

// Do executes a request.
func (cl *Client) Do(ctx context.Context, req *http.Request, result interface{}) error {
	if cl.Jar == nil {
		return errors.New("must supply cookie jar")
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
	if cl.Jar == nil {
		return nil, errors.New("must supply cookie jar")
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.torrentleech.org/download/%d/%s", id, "a"), nil)
	if err != nil {
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
	return ioutil.ReadAll(res.Body)
}

// Option is a TL client option.
type Option func(cl *Client)

// WithJar is an option to set the cookie jar used by the TL client.
func WithJar(jar http.CookieJar) Option {
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
