package test

import (
	"testing"
	"time"

	"github.com/c8121/go_vue_spa/auth"
)

func TestAuthCache_HitRequiresSameUsernameAndPassword(t *testing.T) {
	cache := auth.NewAuthCache(time.Minute)
	user := &auth.AuthenticatedUser{Username: "alice"}
	cache.Add("alice", []byte("secret"), user)

	got, ok := cache.Get("alice", []byte("secret"))
	if !ok || got != user {
		t.Fatalf("expected cache hit with identical credentials, got ok=%v user=%v", ok, got)
	}

	misses := []struct {
		name     string
		username string
		password string
	}{
		{"wrong password", "alice", "wrong"},
		{"empty password", "alice", ""},
		{"other user, same password", "bob", "secret"},
		{"password prefix", "alice", "secre"},
		{"password with trailing space", "alice", "secret "},
	}
	for _, m := range misses {
		t.Run(m.name, func(t *testing.T) {
			if _, ok := cache.Get(m.username, []byte(m.password)); ok {
				t.Fatalf("unexpected cache hit for %q / %q", m.username, m.password)
			}
		})
	}
}

// ("ab", "c") and ("a", "bc") must not produce the same cache key.
func TestAuthCache_KeyIsUnambiguous(t *testing.T) {
	cache := auth.NewAuthCache(time.Minute)
	cache.Add("ab", []byte("c"), &auth.AuthenticatedUser{Username: "ab"})

	if _, ok := cache.Get("a", []byte("bc")); ok {
		t.Fatal("username/password boundary is ambiguous in the cache key")
	}
}

func TestAuthCache_EntriesExpire(t *testing.T) {
	cache := auth.NewAuthCache(100 * time.Millisecond)
	cache.Add("alice", []byte("secret"), &auth.AuthenticatedUser{Username: "alice"})

	if _, ok := cache.Get("alice", []byte("secret")); !ok {
		t.Fatal("expected cache hit before the TTL has passed")
	}

	time.Sleep(400 * time.Millisecond)

	if _, ok := cache.Get("alice", []byte("secret")); ok {
		t.Fatal("expected cache miss after the TTL has passed")
	}
}
