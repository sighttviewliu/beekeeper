package debugapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethersphere/beekeeper"
)

const contentType = "application/json; charset=utf-8"

var userAgent = "beekeeper/" + beekeeper.Version

// Client manages communication with the Bee Debug API.
type Client struct {
	httpClient *http.Client // HTTP client must handle authentication implicitly.
	service    service      // Reuse a single struct instead of allocating one for each service on the heap.

	// Services that API provides.
	Chunks   *ChunksService
	Node     *NodeService
	PingPong *PingPongService
	Postage  *PostageService
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	HTTPClient *http.Client
}

// NewClient constructs a new Client.
func NewClient(baseURL *url.URL, o *ClientOptions) (c *Client) {
	if o == nil {
		o = new(ClientOptions)
	}
	if o.HTTPClient == nil {
		o.HTTPClient = new(http.Client)
	}

	return newClient(httpClientWithTransport(baseURL, o.HTTPClient))
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all API services.
func newClient(httpClient *http.Client) (c *Client) {
	c = &Client{httpClient: httpClient}
	c.service.client = c
	c.Chunks = (*ChunksService)(&c.service)
	c.Node = (*NodeService)(&c.service)
	c.PingPong = (*PingPongService)(&c.service)
	c.Postage = (*PostageService)(&c.service)
	return c
}

func httpClientWithTransport(baseURL *url.URL, c *http.Client) *http.Client {
	if c == nil {
		c = new(http.Client)
	}

	transport := c.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}

	c.Transport = roundTripperFunc(func(r *http.Request) (resp *http.Response, err error) {
		r.Header.Set("User-Agent", userAgent)
		u, err := baseURL.Parse(r.URL.String())
		if err != nil {
			return nil, err
		}
		r.URL = u
		return transport.RoundTrip(r)
	})
	return c
}

// requestJSON handles the HTTP request response cycle. It JSON encodes the request
// body, creates an HTTP request with provided method on a path with required
// headers and decodes request body if the v argument is not nil and content type is
// application/json.
func (c *Client) requestJSON(ctx context.Context, method, path string, body, v interface{}) (err error) {
	var bodyBuffer io.ReadWriter
	if body != nil {
		bodyBuffer = new(bytes.Buffer)
		if err = encodeJSON(bodyBuffer, body); err != nil {
			return err
		}
	}

	return c.request(ctx, method, path, bodyBuffer, v)
}

// request handles the HTTP request response cycle.
func (c *Client) request(ctx context.Context, method, path string, body io.Reader, v interface{}) (err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)

	r, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer drain(r.Body)

	if err = responseErrorHandler(r); err != nil {
		return err
	}

	if v != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return json.NewDecoder(r.Body).Decode(&v)
	}
	return nil
}

// encodeJSON writes a JSON-encoded v object to the provided writer with
// SetEscapeHTML set to false.
func encodeJSON(w io.Writer, v interface{}) (err error) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// drain discards all of the remaining data from the reader and closes it,
// asynchronously.
func drain(r io.ReadCloser) {
	go func() {
		// Panicking here does not put data in
		// an inconsistent state.
		defer func() {
			_ = recover()
		}()

		_, _ = io.Copy(ioutil.Discard, r)
		r.Close()
	}()
}

// responseErrorHandler returns an error based on the HTTP status code or nil if
// the status code is from 200 to 299.
func responseErrorHandler(r *http.Response) (err error) {
	if r.StatusCode/100 == 2 {
		return nil
	}
	switch r.StatusCode {
	case http.StatusBadRequest:
		return decodeBadRequest(r)
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrTooManyRequests
	case http.StatusInternalServerError:
		return ErrInternalServerError
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return errors.New(strings.ToLower(r.Status))
	}
}

// decodeBadRequest parses the body of HTTP response that contains a list of
// errors as the result of bad request data.
func decodeBadRequest(r *http.Response) (err error) {

	type badRequestResponse struct {
		Errors []string `json:"errors"`
	}

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return NewBadRequestError("bad request")
	}
	var e badRequestResponse
	if err = json.NewDecoder(r.Body).Decode(&e); err != nil {
		if err == io.EOF {
			return NewBadRequestError("bad request")
		}
		return err
	}
	return NewBadRequestError(e.Errors...)
}

// service is the base type for all API service providing the Client instance
// for them to use.
type service struct {
	client *Client
}

// Bool is a helper routine that allocates a new bool value to store v and
// returns a pointer to it.
func Bool(v bool) (p *bool) { return &v }

// roundTripperFunc type is an adapter to allow the use of ordinary functions as
// http.RoundTripper interfaces. If f is a function with the appropriate
// signature, roundTripperFunc(f) is a http.RoundTripper that calls f.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(r).
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
