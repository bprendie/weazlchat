package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const userAgent = "weazlchat (https://github.com/bprendie/weazlchat)"

func getJSON(ctx context.Context, client *http.Client, requestURL string, headers map[string]string, out any) error {
	body, _, err := getBody(ctx, client, requestURL, headers, 0)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func getBody(ctx context.Context, client *http.Client, requestURL string, headers map[string]string, maxBytes int64) ([]byte, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	reader := io.Reader(resp.Body)
	if maxBytes > 0 {
		reader = io.LimitReader(resp.Body, maxBytes)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, resp, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, resp, fmt.Errorf("%s returned %s: %s", requestURL, resp.Status, strings.TrimSpace(string(body)))
	}
	return body, resp, nil
}
