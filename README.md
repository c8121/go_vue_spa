# go_vue_spa

Building blocks for serving a **Vue 3 single-page app from a Go binary** (Gin):

- **Embedded frontend base** – Vue 3.5, Vue Router 5, Bootstrap 5.3 and Bootstrap Icons, plus a small modal-dialog helper (`httproot/`)
- **Static file routes** with a default document, served from any `fs.FS` (`routes.go`)
- **Optional HTTP Basic Auth** with a bcrypt user file and a CLI helper to manage users (`auth/`)
- Small helpers for notifications and error handling (`notifications/`, `util/`)

This is a library, not an application: you bring your own `main.go` and `index.html`.

## When to use it

**Good fit**

- **Internal tools and admin UIs** – a dashboard or back-office tool for a small team, shipped as one binary.
- **Self-hosted / home-lab apps** – single-user or few-user services (media library, asset manager, monitoring view) without a separate web server or Node runtime.
- **Prototypes and small products** – start with a working UI + API in one process and no build step (Vue is loaded as ES modules in the browser).
- **Offline or air-gapped environments** – all frontend libraries are embedded, nothing is loaded from a CDN.
- **Tools behind a reverse proxy** – let Caddy/nginx handle TLS and use the built-in Basic Auth as a simple extra layer.

**Not a good fit**

- Public apps with self-registration, SSO/OAuth, sessions or many users – use a proper auth solution instead.
- Apps that need server-side rendering or SEO.
- Large frontends that need a bundler, TypeScript or `.vue` single-file components (these require a build step, which this setup deliberately avoids).

## Quick start

```go
package main

import (
	"embed"

	spa "github.com/c8121/go_vue_spa"
	"github.com/c8121/go_vue_spa/auth"
	"github.com/c8121/go_vue_spa/httproot"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

//go:embed app-ui/*
var appFS embed.FS // your index.html, your JS/CSS

func main() {
	// Optional: enable Basic Auth. Must happen BEFORE CreateRoutesFromFS,
	// otherwise all resources are served without authentication.
	a, err := auth.NewLocalFileAuthenticator("users.txt", bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	if err := a.ReadUsers(); err != nil {
		panic(err)
	}
	auth.SetAuthenticator(a)

	r := gin.Default()

	// The first resource that contains the requested file wins.
	spa.CreateRoutesFromFS(r, []spa.Resource{
		{RootFs: appFS, SubDirName: "app-ui", DefaultDocument: "index.html"},
		{RootFs: httproot.FS, SubDirName: httproot.VueUI},
	})

	// Register your API routes here (they need their own auth handler).

	_ = r.Run(":8080")
}
```

Minimal `app-ui/index.html`:

```html
<link rel="stylesheet" href="/css/bootstrap-5.3.8-dist/css/bootstrap.min.css">
<link rel="stylesheet" href="/css/boostrap-icons-1.13.1/bootstrap-icons.min.css">
<script src="/css/bootstrap-5.3.8-dist/js/bootstrap.bundle.min.js"></script>
<script type="importmap">
{ "imports": {
    "vue": "/js/vue-3.5.41/vue.esm-browser.js",
    "vue-router": "/js/vue-router-5.2.0/vue-router.esm-browser.prod.js"
} }
</script>
<div id="app"></div>
<script type="module" src="/app.js"></script>
```

## Managing users

Users live in a text file (`username<TAB>bcrypt-hash`). Add or edit them with the CLI helper from `auth/cli`
(call `ReadUsers()` first, then e.g. `CliLocalFileUserEdit("alice", true)` to add a user, `false` to change a password).
If the user file does not exist, authentication is enabled but nobody can log in.

## Notes

- Use **hash routing** (`createWebHashHistory`) – only `/` serves the default document, there is no history-mode fallback.
- Basic Auth sends credentials with every request: **run it behind TLS**.
- The dialog helper expects `bootstrap` as a global (loaded via the script tag above).
