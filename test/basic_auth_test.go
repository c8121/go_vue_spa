package test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/c8121/go_vue_spa/auth"
	authgin "github.com/c8121/go_vue_spa/auth/gin"
	"github.com/gin-gonic/gin"
)

// NOTE: The middleware keeps a package-level cache of successful logins (90s).
// Every test therefore uses its own, unique username so that tests cannot
// influence each other through the cache.

// nilUserAuthenticator is a misbehaving custom authenticator: no error, but no user either.
type nilUserAuthenticator struct{}

func (nilUserAuthenticator) Authenticate(string, []byte) (*auth.AuthenticatedUser, error) {
	return nil, nil
}

// countingAuthenticator counts how often the real authenticator is asked.
type countingAuthenticator struct {
	password string
	calls    atomic.Int32
}

func (a *countingAuthenticator) Authenticate(username string, credentials []byte) (*auth.AuthenticatedUser, error) {
	a.calls.Add(1)
	if string(credentials) != a.password {
		return nil, errors.New("invalid credentials")
	}
	return &auth.AuthenticatedUser{Username: username}, nil
}

func newProtectedRouter() *gin.Engine {
	r := gin.New()
	r.GET("/protected", authgin.UserAuthRequiredHandler(func(c *gin.Context) {
		name := ""
		if value, ok := c.Get("user"); ok {
			if user, ok := value.(*auth.AuthenticatedUser); ok {
				name = user.Username
			}
		}
		c.String(http.StatusOK, "hello "+name)
	}))
	return r
}

func requestProtected(t *testing.T, r http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.SetBasicAuth(username, password)
	return serveNoPanic(t, r, req)
}

func TestBasicAuth_MissingCredentialsAreRejected(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"missing-creds-user": "pw"}))
	r := newProtectedRouter()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := serveNoPanic(t, r, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.HasPrefix(w.Header().Get("WWW-Authenticate"), "Basic realm=") {
		t.Fatalf("missing or wrong WWW-Authenticate header: %q", w.Header().Get("WWW-Authenticate"))
	}
}

func TestBasicAuth_ValidCredentialsAreAccepted(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"valid-user": "right"}))

	w := requestProtected(t, newProtectedRouter(), "valid-user", "right")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello valid-user" {
		t.Fatalf("user was not put into the context, body=%q", w.Body.String())
	}
}

func TestBasicAuth_WrongPasswordIsRejected(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"wrongpw-user": "right"}))

	w := requestProtected(t, newProtectedRouter(), "wrongpw-user", "nope")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBasicAuth_UnknownUserIsRejected(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"known-user": "right"}))

	w := requestProtected(t, newProtectedRouter(), "unknown-user", "right")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBasicAuth_CachedLoginDoesNotAcceptOtherPasswords(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"cache-user": "right"}))
	r := newProtectedRouter()

	if w := requestProtected(t, r, "cache-user", "right"); w.Code != http.StatusOK {
		t.Fatalf("first login should succeed, got %d", w.Code)
	}

	for _, pw := range []string{"wrong", "", "righ", "right "} {
		if w := requestProtected(t, r, "cache-user", pw); w.Code != http.StatusUnauthorized {
			t.Fatalf("password %q was accepted after a cached login (status %d)", pw, w.Code)
		}
	}

	if w := requestProtected(t, r, "cache-user", "right"); w.Code != http.StatusOK {
		t.Fatalf("correct password should still work, got %d", w.Code)
	}
}

func TestBasicAuth_SuccessfulLoginsAreCachedFailuresAreNot(t *testing.T) {
	a := &countingAuthenticator{password: "right"}
	installAuthenticator(t, a)
	r := newProtectedRouter()

	for i := 0; i < 3; i++ {
		if w := requestProtected(t, r, "counting-user", "right"); w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}
	if got := a.calls.Load(); got != 1 {
		t.Fatalf("expected 1 call to the authenticator (cache hits afterwards), got %d", got)
	}

	for i := 0; i < 2; i++ {
		if w := requestProtected(t, r, "counting-user", "bad"); w.Code != http.StatusUnauthorized {
			t.Fatalf("bad password: expected 401, got %d", w.Code)
		}
	}
	if got := a.calls.Load(); got != 3 {
		t.Fatalf("failed logins must not be cached: expected 3 calls, got %d", got)
	}
}

// Regression: (nil, nil) from a custom authenticator must not count as "authenticated".
func TestBasicAuth_AuthenticatorReturningNoUserIsRejected(t *testing.T) {
	installAuthenticator(t, nilUserAuthenticator{})

	w := requestProtected(t, newProtectedRouter(), "nil-user", "x")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Regression: malformed headers used to panic ("Basic" without credentials)
// or be misparsed (scheme searched anywhere in the header).
func TestBasicAuth_MalformedHeadersAreRejectedWithoutPanic(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"header-user": "right"}))
	r := newProtectedRouter()

	headers := map[string]string{
		"scheme only":             "Basic",
		"scheme only, lowercase":  "basic",
		"scheme and space":        "Basic ",
		"not base64":              "Basic !!!not-base64!!!",
		"no colon":                "Basic " + base64.StdEncoding.EncodeToString([]byte("nocolon")),
		"other scheme":            "Bearer abc",
		"basic as part of token":  "Bearer basic abc",
		"scheme glued to payload": "BasicXYZ",
		"empty":                   "",
	}

	for name, value := range headers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if value != "" {
				req.Header.Set("Authorization", value)
			}
			w := serveNoPanic(t, r, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 for %q, got %d", value, w.Code)
			}
		})
	}
}

// Documents the CURRENT behaviour: without an authenticator everything is open.
// If you introduce an explicit opt-in (e.g. AllowAnonymous), change this test.
func TestHandler_WithoutAuthenticatorAccessIsOpen(t *testing.T) {
	auth.SetAuthenticator(nil)
	r := newProtectedRouter()

	w := serveNoPanic(t, r, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Known issue: UserAuthRequiredHandler decides ONCE when the handler is created
// whether auth is active. Setting the authenticator afterwards has no effect,
// so the app silently stays open. Remove the Skip once this is fixed.
func TestHandler_AuthenticatorSetAfterWiringStillProtects(t *testing.T) {
	t.Skip("known issue: auth decision is made at wiring time, not per request")

	auth.SetAuthenticator(nil)
	r := newProtectedRouter() // wired BEFORE the authenticator exists
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"late-user": "right"}))

	w := serveNoPanic(t, r, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
