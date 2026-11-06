package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/c8121/go_vue_spa/auth"
	"github.com/c8121/go_vue_spa/notifications"
	"github.com/gin-gonic/gin"

	cache "github.com/go-pkgz/expirable-cache/v3"
)

var (
	notifiedNoUsers = false
	authCache       = cache.NewCache[string, *auth.AuthenticatedUser]().WithTTL(time.Second * 90)
)

func UserAuthRequiredHandler(handlerFunc func(c *gin.Context)) func(c *gin.Context) {

	if auth.GetAuthenticator() == nil {

		if !notifiedNoUsers {
			notifiedNoUsers = true
			notifications.AppNotifications.AddNotification("No auth configured, granting all access")
			slog.Warn("No auth configured, granting all access")
		}

		f := func(c *gin.Context) {
			handlerFunc(c)
		}
		return f
	}

	f := func(c *gin.Context) {
		if err := CheckBasicAuth(c); err != nil {
			slog.Warn("Auth error", "error", err)
			RespondBasicAuthRequired(c)
			return
		}
		handlerFunc(c)
	}

	return f
}

func CheckBasicAuth(c *gin.Context) error {

	username, password, ok := c.Request.BasicAuth()
	if !ok {
		return errors.New("authentication required")
	}

	user, authenticated := authCache.Get(username)
	if authenticated {
		slog.Info("+++++ AUTH")
		c.Set("user", user)
		return nil
	}

	user, err := auth.GetAuthenticator().Authenticate(username, []byte(password))
	if err != nil || user == nil {
		slog.Warn("Auth error", "error", err)
		return err
	}

	c.Set("user", user)
	slog.Info("***** AUTH")
	authCache.Add(username, user)
	return nil
}

func RespondBasicAuthRequired(c *gin.Context) {
	c.Header("WWW-Authenticate", "Basic realm=\"Asset Storage\", charset=\"UTF-8\"")
	c.String(http.StatusUnauthorized, "Unauthorized")
}
