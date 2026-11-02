package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/c8121/go_vue_spa/auth"
	"github.com/c8121/go_vue_spa/notifications"
	"github.com/gin-gonic/gin"
)

var notifiedNoUsers = false

func UserAuthRequiredHandler(handlerFunc func(c *gin.Context)) func(c *gin.Context) {

	if auth.GetAuthenticator() == nil {

		if !notifiedNoUsers {
			notifiedNoUsers = true
			notifications.AppNotifications.AddNotification("No auth configured, granting all access")
		}

		fmt.Println("***WARN*** No auth configured, granting all access")
		f := func(c *gin.Context) {
			handlerFunc(c)
		}
		return f
	}

	f := func(c *gin.Context) {
		if err := CheckBasicAuth(c); err != nil {
			fmt.Printf("Auth error: %s\n", err)
			RespondBasicAuthRequired(c)
			return
		}
		handlerFunc(c)
	}

	return f
}

func CheckBasicAuth(c *gin.Context) error {
	header := c.GetHeader("Authorization")
	if len(header) == 0 {
		return errors.New("authentication required")
	}

	p := strings.Index(strings.ToLower(header), "basic")
	if p < 0 {
		return errors.New("invalid authentication header (only basic supported)")
	}

	key := strings.TrimSpace(header[p+6:])
	decodedKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		fmt.Printf("Basic auth: %s", err)
		return errors.New("cannot decode authentication header")
	}
	key = string(decodedKey)

	p = strings.Index(key, ":")
	if p < 0 {
		fmt.Printf("Basic auth, invalid key: %s, header=%s\n", key, header)
		return errors.New("invalid authentication header")
	}

	username := key[:p]
	password := key[p+1:]

	user, err := auth.GetAuthenticator().Authenticate(username, []byte(password))
	if err != nil || user == nil {
		slog.Warn("Auth error", "error", err)
		return err
	}

	c.Set("user", user)
	return nil
}

func RespondBasicAuthRequired(c *gin.Context) {
	c.Header("WWW-Authenticate", "Basic realm=\"Asset Storage\", charset=\"UTF-8\"")
	c.String(http.StatusUnauthorized, "Unauthorized")
}
