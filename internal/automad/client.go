package automad

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultMaxRetries = 2
	defaultRetryDelay = 250 * time.Millisecond
)

// Client performs authenticated calls against the Automad v2 /_api endpoints.
type Client struct {
	baseURL string
	http    *http.Client
	auth    *authManager

	maxRetries int
	retryDelay time.Duration
}

func newClient(cfg Config) *Client {
	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
		// Do not follow redirects automatically: v2 signals results via a
		// `redirect` field in the JSON body, and following 302s on the login
		// path would drop the Set-Cookie we need.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Client{
		baseURL:    cfg.URL,
		http:       httpClient,
		auth:       newAuthManager(cfg, httpClient),
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
}

// get performs a GET request and returns the unwrapped payload.
func (c *Client) get(ctx context.Context, path string) (any, error) {
	return c.request(ctx, http.MethodGet, path, nil)
}

// post performs a POST request, sending body as the v2 __json__ payload, and
// returns the unwrapped payload.
func (c *Client) post(ctx context.Context, path string, body map[string]any) (any, error) {
	return c.request(ctx, http.MethodPost, path, body)
}

// request runs the authenticated request loop: it attaches the session cookie
// and (for POST) a multipart body carrying __csrf__ and __json__, retrying on
// CSRF mismatch, 401, and v2's "No session" marker by re-authenticating.
func (c *Client) request(ctx context.Context, method, path string, body map[string]any) (any, error) {
	var jsonPayload []byte
	if method == http.MethodPost {
		p, err := json.Marshal(body)
		if err != nil {
			return nil, newError(CodeValidation, "encoding request payload: %v", err)
		}
		jsonPayload = p
	}

	url := c.baseURL + path
	forceReauth := false
	forceRescrape := false
	csrfRescrapes := 0

	for attempt := 0; ; attempt++ {
		cookie, err := c.auth.getCookie(ctx, forceReauth)
		if err != nil {
			return nil, err
		}
		forceReauth = false

		var csrf string
		if method == http.MethodPost {
			csrf, err = c.auth.getCSRF(ctx, forceRescrape)
			if err != nil {
				return nil, err
			}
			forceRescrape = false
		}

		req, err := c.buildRequest(ctx, method, url, csrf, jsonPayload)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, newError(CodeTransport, "%s %s failed: %v", method, path, err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, newError(CodeTransport, "reading %s %s response: %v", method, path, readErr)
		}

		if res.StatusCode == http.StatusOK {
			if attempt < c.maxRetries && envelopeIsNoSession(raw) {
				forceReauth = true
				c.sleep(ctx)
				continue
			}
			return unwrap(raw)
		}

		if res.StatusCode == http.StatusForbidden && attempt < c.maxRetries {
			if msg := envelopeError(raw); csrfMismatchRE.MatchString(msg) {
				if csrfRescrapes >= 1 {
					// A persistent CSRF mismatch after a rescrape means the
					// session itself is dead — re-login rather than rescrape.
					forceReauth = true
					csrfRescrapes = 0
				} else {
					forceRescrape = true
					csrfRescrapes++
				}
				c.sleep(ctx)
				continue
			}
		}

		if res.StatusCode == http.StatusUnauthorized && attempt < c.maxRetries {
			forceReauth = true
			c.sleep(ctx)
			continue
		}

		if res.StatusCode == http.StatusUnauthorized || (res.StatusCode == http.StatusForbidden && csrfRescrapes >= 1) {
			return nil, newError(CodeAuth, "authentication failed: HTTP %d", res.StatusCode)
		}
		return nil, newError(CodeUpstream, "%s %s: HTTP %d", method, path, res.StatusCode)
	}
}

func (c *Client) buildRequest(ctx context.Context, method, url, csrf string, jsonPayload []byte) (*http.Request, error) {
	if method != http.MethodPost {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, newError(CodeTransport, "building request: %v", err)
		}
		return req, nil
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if csrf != "" {
		_ = mw.WriteField("__csrf__", csrf)
	}
	_ = mw.WriteField("__json__", string(jsonPayload))
	if err := mw.Close(); err != nil {
		return nil, newError(CodeTransport, "building multipart body: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, &buf)
	if err != nil {
		return nil, newError(CodeTransport, "building request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req, nil
}

// upload sends a single-chunk Dropzone upload to /_api/file-collection/upload.
// This endpoint skips v2's __json__ decoding, so the destination URL is a plain
// form field and the payload is a real multipart file part.
func (c *Client) upload(ctx context.Context, path, destURL, filename, mimeType string, data []byte) (any, error) {
	url := c.baseURL + path
	forceReauth := false
	forceRescrape := false
	csrfRescrapes := 0

	for attempt := 0; ; attempt++ {
		cookie, err := c.auth.getCookie(ctx, forceReauth)
		if err != nil {
			return nil, err
		}
		forceReauth = false
		csrf, err := c.auth.getCSRF(ctx, forceRescrape)
		if err != nil {
			return nil, err
		}
		forceRescrape = false

		req, err := c.buildUpload(ctx, url, csrf, destURL, filename, mimeType, data)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, newError(CodeTransport, "upload failed: %v", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, newError(CodeTransport, "reading upload response: %v", readErr)
		}

		if res.StatusCode == http.StatusOK {
			if attempt < c.maxRetries && envelopeIsNoSession(raw) {
				forceReauth = true
				c.sleep(ctx)
				continue
			}
			return unwrap(raw)
		}
		if res.StatusCode == http.StatusForbidden && attempt < c.maxRetries && csrfRescrapes < 1 {
			forceRescrape = true
			csrfRescrapes++
			c.sleep(ctx)
			continue
		}
		if res.StatusCode == http.StatusUnauthorized && attempt < c.maxRetries {
			forceReauth = true
			c.sleep(ctx)
			continue
		}
		return nil, newError(CodeUpstream, "upload: HTTP %d", res.StatusCode)
	}
}

func (c *Client) buildUpload(ctx context.Context, url, csrf, destURL, filename, mimeType string, data []byte) (*http.Request, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("__csrf__", csrf)
	if destURL != "" {
		_ = mw.WriteField("url", destURL)
	}
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, newError(CodeTransport, "building upload part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, newError(CodeTransport, "writing upload part: %v", err)
	}
	// Dropzone chunking metadata: a single chunk covering the whole file.
	_ = mw.WriteField("dzuuid", randomID())
	_ = mw.WriteField("dzchunkindex", "0")
	_ = mw.WriteField("dztotalchunkcount", "1")
	_ = mw.WriteField("dzchunkbyteoffset", "0")
	_ = mw.WriteField("dzchunksize", strconv.Itoa(len(data)))
	if err := mw.Close(); err != nil {
		return nil, newError(CodeTransport, "closing upload body: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, newError(CodeTransport, "building upload request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req, nil
}

func (c *Client) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(c.retryDelay):
	}
}

// unwrap decodes v2's {code,data,error,...} envelope. It returns the `data`
// payload on success, or the envelope minus `code` when there is no `data`
// (add/delete/publish answer with redirect/success only), and turns a non-empty
// `error` string into a classified APIError.
func unwrap(raw []byte) (any, error) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, newError(CodeUpstream, "invalid JSON response: %v", err)
	}
	if msg := decodeErrorField(env["error"]); msg != "" {
		return nil, classifyUpstream(msg)
	}
	if d, ok := env["data"]; ok {
		var v any
		if err := json.Unmarshal(d, &v); err != nil {
			return nil, newError(CodeUpstream, "invalid data payload: %v", err)
		}
		if m, ok := v.(map[string]any); ok {
			if msg, _ := m["message"].(string); noSessionRE.MatchString(msg) {
				return nil, newError(CodeAuth, "session is not authenticated")
			}
		}
		return v, nil
	}
	delete(env, "code")
	out := make(map[string]any, len(env))
	for k, r := range env {
		var v any
		if err := json.Unmarshal(r, &v); err == nil {
			out[k] = v
		}
	}
	return out, nil
}

// envelopeError returns the non-empty `error` string of an envelope, or "".
func envelopeError(raw []byte) string {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil {
		return ""
	}
	return decodeErrorField(env["error"])
}

// envelopeIsNoSession reports whether the envelope carries v2's "No session"
// marker in data.message, signalling an expired session.
func envelopeIsNoSession(raw []byte) bool {
	var env struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return false
	}
	return noSessionRE.MatchString(env.Data.Message)
}

func decodeErrorField(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(r, &s); err != nil {
		return ""
	}
	return s
}

// classifyUpstream maps v2's terse error strings to actionable codes.
func classifyUpstream(msg string) *APIError {
	return newError(CodeUpstream, "%s", msg)
}

// randomID returns a random hex identifier for the Dropzone upload contract.
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}
