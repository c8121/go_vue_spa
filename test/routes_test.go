package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	spa "github.com/c8121/go_vue_spa"
	"github.com/gin-gonic/gin"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"app/index.html":      {Data: []byte("<h1>app</h1>")},
		"app/js/app.js":       {Data: []byte("console.log('app')")},
		"app/js/shared.js":    {Data: []byte("app-version")},
		"common/js/shared.js": {Data: []byte("common-version")},
		"common/js/common.js": {Data: []byte("common-only")},
		"secret/secret.txt":   {Data: []byte("TOP SECRET")},
	}
}

func newSpaRouter(resources []spa.Resource) *gin.Engine {
	r := gin.New()
	spa.CreateRoutesFromFS(r, resources)
	return r
}

func getPath(t *testing.T, r http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, target, nil)
	return serveNoPanic(t, r, req)
}

func appResource(fsys fstest.MapFS) spa.Resource {
	return spa.Resource{RootFs: fsys, SubDirName: "app", DefaultDocument: "index.html"}
}

func TestRoutes_ServesDefaultDocumentAtRoot(t *testing.T) {
	r := newSpaRouter([]spa.Resource{appResource(testFS())})

	w := getPath(t, r, "/")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<h1>app</h1>") {
		t.Fatalf("unexpected body: %q", w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("unexpected content type: %q", w.Header().Get("Content-Type"))
	}
}

func TestRoutes_ServesFilesFromSubDir(t *testing.T) {
	r := newSpaRouter([]spa.Resource{appResource(testFS())})

	for _, target := range []string{"/js/app.js", "/js/app.js?cache=1"} {
		w := getPath(t, r, target)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", target, w.Code)
		}
		if w.Body.String() != "console.log('app')" {
			t.Fatalf("%s: unexpected body %q", target, w.Body.String())
		}
	}
}

func TestRoutes_UnknownFilesAndDirectoriesAreNotFound(t *testing.T) {
	r := newSpaRouter([]spa.Resource{appResource(testFS())})

	for _, target := range []string{"/nope.js", "/js", "/js/", "/js/missing.js"} {
		if w := getPath(t, r, target); w.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", target, w.Code)
		}
	}
}

func TestRoutes_PathTraversalIsBlocked(t *testing.T) {
	r := newSpaRouter([]spa.Resource{appResource(testFS())})

	targets := []string{
		"/../secret/secret.txt",
		"/js/../../secret/secret.txt",
		"/%2e%2e/secret/secret.txt",
		"/..%2fsecret/secret.txt",
		"/js/..%2f..%2fsecret/secret.txt",
	}
	for _, target := range targets {
		w := getPath(t, r, target)
		if strings.Contains(w.Body.String(), "TOP SECRET") {
			t.Fatalf("%s: leaked content outside of the resource directory", target)
		}
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", target, w.Code)
		}
	}
}

func TestRoutes_FirstMatchingResourceWins(t *testing.T) {
	fsys := testFS()
	r := newSpaRouter([]spa.Resource{
		appResource(fsys),
		{RootFs: fsys, SubDirName: "common"},
	})

	if w := getPath(t, r, "/js/shared.js"); w.Body.String() != "app-version" {
		t.Fatalf("first resource should win, got %q", w.Body.String())
	}
	if w := getPath(t, r, "/js/common.js"); w.Code != http.StatusOK || w.Body.String() != "common-only" {
		t.Fatalf("file only in the second resource should be served, got %d %q", w.Code, w.Body.String())
	}
}

func TestRoutes_DefaultDocumentIsTakenFromTheFirstResourceThatHasOne(t *testing.T) {
	fsys := testFS()
	r := newSpaRouter([]spa.Resource{
		{RootFs: fsys, SubDirName: "common"}, // no default document
		appResource(fsys),
	})

	w := getPath(t, r, "/")

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "<h1>app</h1>") {
		t.Fatalf("expected default document of the second resource, got %d %q", w.Code, w.Body.String())
	}
}

func TestRoutes_NoDefaultDocumentMeansNotFoundAtRoot(t *testing.T) {
	r := newSpaRouter([]spa.Resource{{RootFs: testFS(), SubDirName: "common"}})

	if w := getPath(t, r, "/"); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRoutes_AreProtectedWhenAuthIsConfigured(t *testing.T) {
	installAuthenticator(t, newUserAuthenticator(t, map[string]string{"routes-user": "right"}))
	// The authenticator must exist BEFORE the routes are created (see basic_auth_test.go).
	r := newSpaRouter([]spa.Resource{appResource(testFS())})

	for _, target := range []string{"/", "/js/app.js", "/nope.js"} {
		w := getPath(t, r, target)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s without credentials: expected 401, got %d", target, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/js/app.js", nil)
	req.SetBasicAuth("routes-user", "right")
	if w := serveNoPanic(t, r, req); w.Code != http.StatusOK {
		t.Fatalf("with credentials: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/js/app.js", nil)
	req.SetBasicAuth("routes-user", "wrong")
	if w := serveNoPanic(t, r, req); w.Code != http.StatusUnauthorized {
		t.Fatalf("with wrong password: expected 401, got %d", w.Code)
	}
}
