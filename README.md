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

	// Register your API routes here (see "Protecting API routes" below).

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

Request paths are resolved inside the resource's `SubDirName` only; `..` segments cannot escape it.
Directories are never listed.

## Authentication

Authentication is HTTP Basic Auth against an `auth.Authenticator`. The built-in
`LocalFileAuthenticator` stores users in a text file (`username<TAB>bcrypt-hash`).

- **User names** are case-insensitive and whitespace is ignored (`Alice`, ` alice ` and `ALICE` are the same user).
- **User file:** lines starting with `#` and invalid lines are ignored. Saving writes a temp file and renames it
  (atomic), with mode `0600`.
- **No user file** (or no users) means authentication is enabled but nobody can log in.
- **No authenticator set** means everything is served **without** authentication (a warning is logged).
- **Login cache:** successful logins are cached for 90 seconds (keyed by user name *and* password), so a page
  load does not run bcrypt for every asset. Changed passwords or removed users therefore take up to 90 seconds
  to take effect. Failed logins are never cached.
- **Realm:** set `authgin.BasicAuthRealm` (default `"SPA"`).

### Protecting API routes

Only the static file handler is protected automatically. Wrap your own handlers:

```go
import (
	"github.com/c8121/go_vue_spa/auth"
	authgin "github.com/c8121/go_vue_spa/auth/gin"
)

r.GET("/api/items", authgin.UserAuthRequiredHandler(func(c *gin.Context) {
	user, _ := c.Get("user") // *auth.AuthenticatedUser
	c.JSON(200, gin.H{"user": user.(*auth.AuthenticatedUser).Username})
}))
```

### Custom authenticators

Implement `auth.Authenticator` (`Authenticate(username string, credentials []byte) (*AuthenticatedUser, error)`)
and register it with `auth.SetAuthenticator`. Return an error on failure; returning `(nil, nil)` is treated as
a failed login.

## Managing users

Users are managed with the CLI helper in `auth/cli`. It needs the `LocalFileAuthenticator` to be registered
via `auth.SetAuthenticator` and the existing users loaded via `ReadUsers()`:

```go
import authcli "github.com/c8121/go_vue_spa/auth/cli"

authcli.CliLocalFileUserEdit("alice", true)  // add user (prompts for the password)
authcli.CliLocalFileUserEdit("alice", false) // change password
```

## Dialog helper

`httproot/vue-ui/js/dialog.js` creates Bootstrap modals whose body is a Vue component. It expects `bootstrap`
as a global (loaded via the script tag above) and the `vue` import map entry.

```js
import dialog from '/js/dialog.js'

const dlg = dialog.create()
dlg.setTitle('Rename')
dlg.setConfirmText('Save')
dlg.setContentComponent({
    template: `<input class="form-control" v-model="name">`,
    data() { return {name: ''} }
})
dlg.addOnConfirmListener(() => console.log('confirmed'))
```

There is intentionally no method to set raw HTML as dialog content (XSS risk); use a component.

## Notes and limitations

- Use **hash routing** (`createWebHashHistory`) – only `/` serves the default document, there is no history-mode fallback.
- Basic Auth sends credentials with every request: **run it behind TLS**.
- There is no rate limiting or lockout for failed logins; put the app behind a reverse proxy if it is exposed.
- All HTTP methods are answered by the static file handler.
- The embedded Bootstrap/Vue files include source maps and unminified variants, so the binary is about 12 MB larger
  than strictly necessary.

## Development

```
go test -race ./test/...
```

Note: the login cache is package-level state, so tests that log in should use unique user names.

## License

MIT, see [LICENSE](LICENSE). The bundled Vue, Vue Router, Bootstrap and Bootstrap Icons are MIT-licensed as well.
