package test

import (
	"os"
	"testing"

	"github.com/c8121/go_vue_spa/auth"
)

func TestLocalFileAuthenticator(t *testing.T) {

	f, err := os.CreateTemp("", "test-users")
	if err != nil {
		t.Fatal(err)
	}

	authenticator, err := auth.NewLocalFileAuthenticator(f.Name(), 1)
	if err != nil {
		t.Fatal(err)
	}

	err = authenticator.AddOrUpdateUser("test", []byte("test"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = authenticator.Authenticate("test", []byte("test"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = authenticator.Authenticate("test-false", []byte("test"))
	if err == nil {
		t.Fatal(err)
	}

	_, err = authenticator.Authenticate("test", []byte("test-false"))
	if err == nil {
		t.Fatal(err)
	}

	_, err = authenticator.Authenticate("test-false", []byte("test-false"))
	if err == nil {
		t.Fatal(err)
	}

	err = authenticator.SaveUsers()
	if err != nil {
		t.Fatal(err)
	}

	authLoaded, err := auth.NewLocalFileAuthenticator(f.Name(), 1)
	if err != nil {
		t.Fatal(err)
	}

	err = authLoaded.ReadUsers()
	if err != nil {
		t.Fatal(err)
	}

	_, err = authLoaded.Authenticate("test", []byte("test"))
	if err != nil {
		t.Fatal(err)
	}

	_ = os.Remove(f.Name())
}
