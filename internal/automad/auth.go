package automad

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// csrfMetaRE extracts the CSRF token from the dashboard HTML. Automad renders
// `<meta name="csrf" content="<64-hex>">`; the two orderings below tolerate the
// attributes appearing in either sequence.
var (
	csrfMetaRE     = regexp.MustCompile(`(?i)<meta\s+name=["']csrf["']\s+content=["']([a-f0-9]{64})["']`)
	csrfMetaAltRE  = regexp.MustCompile(`(?i)<meta\s+content=["']([a-f0-9]{64})["']\s+name=["']csrf["']`)
	sessionCookie  = regexp.MustCompile(`(?i)^Automad-[a-f0-9]+$`)
	noSessionRE    = regexp.MustCompile(`(?i)^\s*No session\s*$`)
	csrfMismatchRE = regexp.MustCompile(`(?i)csrf`)
)

// extractCSRF pulls the CSRF token out of dashboard HTML, or "" if absent.
func extractCSRF(html string) string {
	if m := csrfMetaRE.FindStringSubmatch(html); m != nil {
		return strings.ToLower(m[1])
	}
	if m := csrfMetaAltRE.FindStringSubmatch(html); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// authManager owns the session cookie and CSRF token, re-authenticating and
// re-scraping on demand. It is safe for concurrent use.
type authManager struct {
	cfg  Config
	http *http.Client

	mu     sync.Mutex
	cookie string // full "name=value" pair
	csrf   string
}

func newAuthManager(cfg Config, client *http.Client) *authManager {
	return &authManager{cfg: cfg, http: client}
}

// getCookie returns the session cookie, logging in when needed or forced.
func (a *authManager) getCookie(ctx context.Context, force bool) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !force && a.cookie != "" {
		return a.cookie, nil
	}
	if err := a.login(ctx); err != nil {
		return "", err
	}
	return a.cookie, nil
}

// getCSRF returns the CSRF token, scraping a fresh one when needed or forced.
func (a *authManager) getCSRF(ctx context.Context, force bool) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !force && a.csrf != "" {
		return a.csrf, nil
	}
	if a.cookie == "" {
		if err := a.login(ctx); err != nil {
			return "", err
		}
		return a.csrf, nil
	}
	if err := a.scrapeCSRF(ctx); err != nil {
		// A forced refresh that fails likely means the session died; re-login.
		if force {
			if err := a.login(ctx); err != nil {
				return "", err
			}
			return a.csrf, nil
		}
		return "", err
	}
	return a.csrf, nil
}

// login authenticates against /_api/session/login and scrapes an initial CSRF
// token. v2 returns 200 with a session cookie for any input, so a successful
// HTTP response is not proof of valid credentials — a rejected login carries an
// `error` key, and the cookie is still anonymous. We surface both.
func (a *authManager) login(ctx context.Context) error {
	loginURL := a.cfg.URL + APIBase + "/session/login"
	form := url.Values{
		"name-or-email": {a.cfg.Username},
		"password":      {a.cfg.Password},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return newError(CodeTransport, "building login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := a.http.Do(req)
	if err != nil {
		return newError(CodeTransport, "login request failed: %v", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return newError(CodeAuth, "login failed: HTTP %d", res.StatusCode)
	}
	if msg := envelopeError(body); msg != "" {
		return newError(CodeAuth, "login rejected: %s", msg)
	}

	cookie := sessionCookieFrom(res.Cookies())
	if cookie == "" {
		return newError(CodeAuth, "no session cookie returned by Automad login")
	}
	a.cookie = cookie

	if err := a.scrapeCSRF(ctx); err != nil {
		a.cookie = ""
		return err
	}
	return nil
}

// scrapeCSRF fetches the dashboard and extracts the CSRF meta token, adopting a
// rotated session cookie if v2 issued one.
func (a *authManager) scrapeCSRF(ctx context.Context) error {
	// The http.Client is configured not to follow redirects (to preserve the
	// login Set-Cookie), but /dashboard 301-redirects to /dashboard/setup, so
	// follow the hops manually, carrying and adopting the session cookie.
	next := a.cfg.URL + "/dashboard"
	for range 5 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			return newError(CodeTransport, "building dashboard request: %v", err)
		}
		if a.cookie != "" {
			req.Header.Set("Cookie", a.cookie)
		}
		res, err := a.http.Do(req)
		if err != nil {
			return newError(CodeTransport, "dashboard request failed: %v", err)
		}
		if rotated := sessionCookieFrom(res.Cookies()); rotated != "" {
			a.cookie = rotated
		}
		if loc := res.Header.Get("Location"); res.StatusCode >= 300 && res.StatusCode < 400 && loc != "" {
			res.Body.Close()
			next = resolveLocation(a.cfg.URL, loc)
			continue
		}
		html, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return newError(CodeAuth, "fetching dashboard for CSRF failed: HTTP %d", res.StatusCode)
		}
		if readErr != nil {
			return newError(CodeTransport, "reading dashboard body: %v", readErr)
		}
		token := extractCSRF(string(html))
		if token == "" {
			return newError(CodeAuth, "could not extract CSRF token from dashboard HTML")
		}
		a.csrf = token
		return nil
	}
	return newError(CodeAuth, "too many redirects fetching dashboard for CSRF")
}

// resolveLocation resolves a redirect Location against the instance base URL,
// handling both absolute and root-relative targets.
func resolveLocation(base, loc string) string {
	if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
		return loc
	}
	if strings.HasPrefix(loc, "/") {
		return base + loc
	}
	return base + "/" + loc
}

// sessionCookieFrom returns the first Automad session cookie as "name=value".
func sessionCookieFrom(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if sessionCookie.MatchString(c.Name) {
			return fmt.Sprintf("%s=%s", c.Name, c.Value)
		}
	}
	return ""
}
