package test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/c8121/go_vue_spa/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestAuthenticator creates an authenticator with a (not yet existing) user file in a temp dir.
// bcrypt.MinCost keeps the tests fast.
func newTestAuthenticator(t *testing.T) (*auth.LocalFileAuthenticator, string) {
	t.Helper()

	file := filepath.Join(t.TempDir(), "users.txt")
	a, err := auth.NewLocalFileAuthenticator(file, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("NewLocalFileAuthenticator: %v", err)
	}
	return a, file
}

func mustAddUser(t *testing.T, a *auth.LocalFileAuthenticator, username, password string) {
	t.Helper()

	if err := a.AddOrUpdateUser(username, []byte(password)); err != nil {
		t.Fatalf("AddOrUpdateUser(%q): %v", username, err)
	}
}

func newUserAuthenticator(t *testing.T, users map[string]string) *auth.LocalFileAuthenticator {
	t.Helper()

	a, _ := newTestAuthenticator(t)
	for username, password := range users {
		mustAddUser(t, a, username, password)
	}
	return a
}

// installAuthenticator sets the global authenticator and removes it again after the test.
func installAuthenticator(t *testing.T, a auth.Authenticator) {
	t.Helper()

	auth.SetAuthenticator(a)
	t.Cleanup(func() { auth.SetAuthenticator(nil) })
}

// serveNoPanic serves the request and reports a panic in the handler as a test failure
// instead of crashing the whole test binary.
func serveNoPanic(t *testing.T, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handler panicked: %v", r)
		}
	}()
	h.ServeHTTP(w, req)
	return w
}
