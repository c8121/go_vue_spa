package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	cache "github.com/go-pkgz/expirable-cache/v3"
)

type AuthCache struct {
	cache cache.Cache[string, *AuthenticatedUser]
}

func NewAuthCache(ttl time.Duration) *AuthCache {
	return &AuthCache{
		cache: cache.NewCache[string, *AuthenticatedUser]().WithTTL(ttl),
	}
}

func (ac *AuthCache) Add(username string, password []byte) {
	key := ac.createCacheKey(username, password)
	ac.cache.Add(key, &AuthenticatedUser{})
}

func (ac *AuthCache) Get(username string, password []byte) (*AuthenticatedUser, bool) {
	key := ac.createCacheKey(username, password)
	return ac.cache.Get(key)
}

func (ac *AuthCache) createCacheKey(username string, password []byte) string {
	h := sha256.New()
	h.Write([]byte(username))
	h.Write([]byte{0})
	h.Write(password)

	var sum [sha256.Size]byte
	h.Sum(sum[:0])

	return hex.EncodeToString(sum[:])
}
