package flightradar

import "net/http"

const platformVersion = "25.197.0927"

// defaultGRPCHeaders builds the baseline headers for FR24 gRPC-web.
func defaultGRPCHeaders(deviceID string, bearer string) http.Header {
	h := make(http.Header)
	for k, vs := range defaultJSONHeaders(deviceID) {
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	h.Set("Accept", "*/*")
	h.Set("fr24-platform", "web-"+platformVersion)
	h.Set("x-envoy-retry-grpc-on", "unavailable")
	h.Set("Content-Type", "application/grpc-web+proto")
	h.Set("X-User-Agent", "grpc-web-javascript/0.1")
	h.Set("X-Grpc-Web", "1")
	h.Set("DNT", "1")
	if bearer != "" {
		h.Set("authorization", "Bearer "+bearer)
	}
	// device id header
	if deviceID != "" {
		h.Set("fr24-device-id", deviceID)
	}
	return h
}
