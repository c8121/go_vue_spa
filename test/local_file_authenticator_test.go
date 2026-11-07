package test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/c8121/go_vue_spa/auth"
	"golang.org/x/crypto/bcrypt"
)

func requireAuthOK(t *testing.T, a *auth.LocalFileAuthenticator, username, password string) *auth.AuthenticatedUser {
	t.Helper()

	user, err := a.Authenticate(username, []byte(password))
	if err != nil {
		t.Fatalf("Authenticate(%q) should succeed: %v", username, err)
	}
	if user == nil {
		t.Fatalf("Authenticate(%q) returned nil user without error", username)
	}
	return user
}

func requireAuthFails(t *testing.T, a *auth.LocalFileAuthenticator, username, password string) {
	t.Helper()

	user, err := a.Authenticate(username, []byte(password))
	if err == nil {
		t.Fatalf("Authenticate(%q) should fail but returned user %+v", username, user)
	}
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestLocalFile_UserNamesAreCaseInsensitive(t *testing.T) {
	a, _ := newTestAuthenticator(t)
	mustAddUser(t, a, "Alice", "pw")

	user := requireAuthOK(t, a, "ALICE", "pw")
	if user.Username != "alice" {
		t.Fatalf("expected normalized username %q, got %q", "alice", user.Username)
	}
	if !a.UserExists("aLiCe") {
		t.Fatal("UserExists should ignore case")
	}
}

func TestLocalFile_SurroundingWhitespaceIsIgnored(t *testing.T) {
	a, _ := newTestAuthenticator(t)
	mustAddUser(t, a, "  bob  ", "pw")

	requireAuthOK(t, a, "bob", "pw")
	requireAuthOK(t, a, "\tbob\n", "pw")
}

func TestLocalFile_InnerWhitespaceIsRemoved(t *testing.T) {
	a, _ := newTestAuthenticator(t)
	mustAddUser(t, a, "Al ice", "pw")

	requireAuthOK(t, a, "alice", "pw")
	requireAuthOK(t, a, "A L I C E", "pw")
}

func TestLocalFile_EmptyUserNameIsRejected(t *testing.T) {
	a, _ := newTestAuthenticator(t)

	for _, name := range []string{"", "   ", "\t\n"} {
		if err := a.AddOrUpdateUser(name, []byte("pw")); err == nil {
			t.Fatalf("AddOrUpdateUser(%q) should fail", name)
		}
	}
}

func TestLocalFile_UpdateUserChangesPassword(t *testing.T) {
	a, _ := newTestAuthenticator(t)
	mustAddUser(t, a, "carol", "old")
	mustAddUser(t, a, "carol", "new")

	requireAuthFails(t, a, "carol", "old")
	requireAuthOK(t, a, "carol", "new")
}

func TestLocalFile_ControlCharactersDoNotCorruptUserFile(t *testing.T) {
	a, file := newTestAuthenticator(t)
	mustAddUser(t, a, "a\tb", "pw1")
	mustAddUser(t, a, "x\ny", "pw2")

	if err := a.SaveUsers(); err != nil {
		t.Fatalf("SaveUsers: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if lines := nonEmptyLines(string(data)); len(lines) != 2 {
		t.Fatalf("expected exactly 2 lines in user file, got %d:\n%s", len(lines), data)
	}

	reloaded, err := auth.NewLocalFileAuthenticator(file, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.ReadUsers(); err != nil {
		t.Fatalf("ReadUsers: %v", err)
	}
	requireAuthOK(t, reloaded, "ab", "pw1")
	requireAuthOK(t, reloaded, "xy", "pw2")
}

func TestLocalFile_SaveAndReloadRoundtrip(t *testing.T) {
	a, file := newTestAuthenticator(t)
	users := map[string]string{"alice": "pw-a", "bob": "pw-b", "carol": "pw-c"}
	for name, pw := range users {
		mustAddUser(t, a, name, pw)
	}
	if err := a.SaveUsers(); err != nil {
		t.Fatalf("SaveUsers: %v", err)
	}

	reloaded, err := auth.NewLocalFileAuthenticator(file, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.ReadUsers(); err != nil {
		t.Fatalf("ReadUsers: %v", err)
	}
	for name, pw := range users {
		requireAuthOK(t, reloaded, name, pw)
		requireAuthFails(t, reloaded, name, pw+"x")
	}
}

func TestLocalFile_ReadUsersHandlesCRLFCommentsAndInvalidLines(t *testing.T) {
	a, file := newTestAuthenticator(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	content := "# users\r\n" +
		"\r\n" +
		"no-tab-in-this-line\r\n" +
		"\tmissing-username\r\n" +
		"alice\t" + string(hash) + "\r\n"
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	if err := a.ReadUsers(); err != nil {
		t.Fatalf("ReadUsers: %v", err)
	}

	requireAuthOK(t, a, "alice", "pw") // would fail if "\r" stayed in the hash
	if a.UserExists("no-tab-in-this-line") || a.UserExists("missing-username") {
		t.Fatal("invalid lines must be ignored")
	}
}

func TestLocalFile_MissingUserFileLocksEverything(t *testing.T) {
	a, _ := newTestAuthenticator(t)

	if err := a.ReadUsers(); err != nil {
		t.Fatalf("ReadUsers on a missing file should not fail: %v", err)
	}
	requireAuthFails(t, a, "anyone", "anything")
}

func TestLocalFile_RejectsNonRegularUserFile(t *testing.T) {
	if _, err := auth.NewLocalFileAuthenticator(t.TempDir(), bcrypt.MinCost); err == nil {
		t.Fatal("a directory must not be accepted as user file")
	}
}

func TestLocalFile_SaveUsersUsesRestrictivePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions are not available on Windows")
	}

	a, file := newTestAuthenticator(t)
	mustAddUser(t, a, "alice", "pw")
	if err := a.SaveUsers(); err != nil {
		t.Fatalf("SaveUsers: %v", err)
	}

	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected user file mode 0600, got %o", perm)
	}
}

func TestLocalFile_SaveUsersLeavesNoTempFiles(t *testing.T) {
	a, file := newTestAuthenticator(t)
	mustAddUser(t, a, "alice", "pw")
	for i := 0; i < 3; i++ {
		if err := a.SaveUsers(); err != nil {
			t.Fatalf("SaveUsers: %v", err)
		}
	}

	entries, err := os.ReadDir(filepath.Dir(file))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(file) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected only the user file, found %v", names)
	}
}

func TestLocalFile_SaveUsersReportsFailure(t *testing.T) {
	// The directory does not exist, so the temp file cannot be created.
	file := filepath.Join(t.TempDir(), "missing-dir", "users.txt")
	a, err := auth.NewLocalFileAuthenticator(file, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	mustAddUser(t, a, "alice", "pw")

	if err := a.SaveUsers(); err == nil {
		t.Fatal("SaveUsers must return an error when the file cannot be written")
	}
}

// Run with `go test -race` to detect data races on the user map.
func TestLocalFile_ConcurrentUse(t *testing.T) {
	a, _ := newTestAuthenticator(t)
	mustAddUser(t, a, "base", "pw")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("user%d", i)
			for j := 0; j < 5; j++ {
				if err := a.AddOrUpdateUser(name, []byte("pw")); err != nil {
					t.Error(err)
					return
				}
				_ = a.UserExists(name)
				_, _ = a.Authenticate("base", []byte("pw"))
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 5; j++ {
			if err := a.SaveUsers(); err != nil {
				t.Error(err)
				return
			}
		}
	}()

	wg.Wait()
}
