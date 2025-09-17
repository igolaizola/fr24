package flightradar

import (
	"context"
	"net/http"
	"time"
)

// Client is the root entrypoint. It manages headers and optional auth and
// exposes high-level JSON API calls (flight list, airport list, playback, find).
//
// Note: gRPC methods are not included in this MVP to avoid requiring protoc.
type Client struct {
	http    *http.Client
	headers http.Header
	// subscriptionKey, when set, is sent as "token" query parameter.
	subscriptionKey string
	// deviceID is sent as "fr24-device-id" header when unauthenticated.
	deviceID string
	// authToken (Bearer) for gRPC-web endpoints when logged in with username/password.
	authToken string
}

// New creates a Client with sane defaults and a short timeout.
func New() *Client {
	return &Client{
		http:     &http.Client{Timeout: 20 * time.Second},
		headers:  http.Header(defaultJSONHeaders("")),
		deviceID: newDeviceID(),
	}
}

// WithHTTP replaces the underlying http.Client.
func (c *Client) WithHTTP(h *http.Client) *Client {
	if h != nil {
		c.http = h
	}
	return c
}

// WithSubscriptionKey sets the FR24 subscription key used by JSON endpoints.
func (c *Client) WithSubscriptionKey(key string) *Client {
	c.subscriptionKey = key
	return c
}

// WithDeviceID overrides the random device ID header used for anonymous access.
func (c *Client) WithDeviceID(id string) *Client {
	if id != "" {
		c.deviceID = id
		c.headers.Set("fr24-device-id", id)
	}
	return c
}

// do executes a request with base headers and context.
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// ensure a context
	if ctx == nil {
		ctx = context.Background()
	}
	req = req.WithContext(ctx)
	// merge headers
	for k, vals := range c.headers {
		for _, v := range vals {
			if req.Header.Get(k) == "" {
				req.Header.Add(k, v)
			}
		}
	}
	// default device id header if not set
	if req.Header.Get("fr24-device-id") == "" {
		req.Header.Set("fr24-device-id", c.deviceID)
	}
	return c.http.Do(req)
}
