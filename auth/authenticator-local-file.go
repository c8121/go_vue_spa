package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/c8121/go_vue_spa/util"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultFilePermissions = 0600
)

type LocalFileAuthenticator struct {
	userFile        string
	filePermissions int
	pwdCost         int

	users *UserMap
}

func NewLocalFileAuthenticator(userFile string, pwdCost int) (*LocalFileAuthenticator, error) {

	stat, err := os.Stat(userFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		//User file does not exist... use app with auth but no users (locked)
	} else if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", userFile)
	}

	auth := &LocalFileAuthenticator{
		userFile:        userFile,
		filePermissions: DefaultFilePermissions,
		users:           NewUserMap(),
		pwdCost:         pwdCost,
	}

	return auth, nil
}

type UserMap struct {
	users map[string]string
	mutex sync.Mutex
}

func NewUserMap() *UserMap {
	users := make(map[string]string)
	return &UserMap{
		users: users,
	}
}

func (u *UserMap) GetHash(username string) (string, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	hash, ok := u.users[username]
	return hash, ok
}

func (u *UserMap) SetHash(username string, hash string) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.users[username] = hash
}

func (u *UserMap) Walk(handler func(username string, hash string) error) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	for n, h := range u.users {
		err := handler(n, h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *LocalFileAuthenticator) Authenticate(username string, credentials []byte) (*AuthenticatedUser, error) {
	if !a.UserExists(username) {
		return nil, fmt.Errorf("user does not exist")
	}

	username, err := a.normalizeUserName(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	pwdHash, _ := a.users.GetHash(username)
	err = bcrypt.CompareHashAndPassword([]byte(pwdHash), credentials)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	return &AuthenticatedUser{Username: username}, nil
}

func createPasswordHash(password []byte, pwdCost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(password, pwdCost)
	if err != nil {
		fmt.Printf("Cannot create hash: %s\n", err)
		return "", err
	}

	return string(hash), nil
}

func (a *LocalFileAuthenticator) UserExists(username string) bool {
	if len(username) == 0 {
		return false
	}

	username, err := a.normalizeUserName(username)
	if err != nil {
		slog.Warn("Invalid username", "error", err)
		return false
	}

	_, ok := a.users.GetHash(username)
	return ok
}

func (a *LocalFileAuthenticator) AddOrUpdateUser(username string, password []byte) error {

	if len(username) == 0 {
		return errors.New("empty user name not allowed")
	}

	pwdHash, err := createPasswordHash(password, a.pwdCost)
	if err != nil {
		return err
	}

	username, err = a.normalizeUserName(username)
	if err != nil {
		return err
	}

	_, ok := a.users.GetHash(username)
	if ok {
		slog.Info("Update user", "username", username)
	} else {
		slog.Info("Add new user", "username", username)
	}
	a.users.SetHash(username, pwdHash)
	return nil
}

func (a *LocalFileAuthenticator) ReadUsers() error {

	_, err := os.Stat(a.userFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil //File doesn't exist yet, nothing to read
	}

	f, err := os.Open(a.userFile)
	if err != nil {
		return err
	}
	defer util.CloseOrLog(f)

	reader := bufio.NewReaderSize(f, 1000)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		s := string(line)
		p := strings.Index(s, "\t")
		if p < 1 {
			slog.Warn("Invalid line", "content", s)
			continue
		}

		username, err := a.normalizeUserName(s[0:p])
		if err != nil {
			slog.Warn("Invalid line", "error", err)
			continue
		}

		a.users.SetHash(username, strings.TrimSpace(s[p+1:]))
	}
	return nil
}

func (a *LocalFileAuthenticator) SaveUsers() error {

	f, err := os.CreateTemp(filepath.Dir(a.userFile), filepath.Base(a.userFile))
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	err = a.users.Walk(func(username string, hash string) error {
		_, err := w.WriteString(username + "\t" + hash + "\n")
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		util.CloseOrLog(f)
		return err
	}

	err = w.Flush()
	if err != nil {
		util.CloseOrLog(f)
		return err
	}
	util.CloseOrLog(f)

	err = os.Rename(f.Name(), a.userFile)
	if err != nil {
		slog.Warn("Failed to rename", "file", f.Name(), "dest", a.userFile, "error", err)
		return err
	}

	err = os.Chmod(a.userFile, os.FileMode(a.filePermissions))
	if err != nil {
		slog.Warn("Failed to change permissions", "file", a.userFile, "error", err)
	}

	slog.Info("Updated user file", "path", a.userFile)
	return nil
}

// normalizeUserName removes all whitespace (even contained username) and converts to lower case.
func (a *LocalFileAuthenticator) normalizeUserName(username string) (string, error) {
	username = strings.TrimSpace(username)
	strings.Join(strings.Fields(username), "")
	username = strings.ToLower(username)

	if len(username) == 0 {
		return username, fmt.Errorf("username is empty")
	}

	return username, nil
}
