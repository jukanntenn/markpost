// Package httputil provides HTTP helper functions for making requests and checking responses.
package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

// CheckHTTPResponse returns an error if the response status is outside the 2xx range.
func CheckHTTPResponse(resp *http.Response, path string) error {
	if !isSuccessStatus(resp.StatusCode) {
		return fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}
	return nil
}

// FetchAndDecodeJSON performs an HTTP GET to the given URL, checks the response
// status, and decodes the JSON body into target. The caller is responsible for
// closing the response body.
func FetchAndDecodeJSON(client *http.Client, url string, target any) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := CheckHTTPResponse(resp, url); err != nil {
		return err
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
