package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/c8121/go_vue_spa/util"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultFilePermissions = 0700
)

type LocalFileAuthenticator struct {
	userFile        string
	filePermissions int
	pwdCost         int

	users map[string]string
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
		users:           make(map[string]string),
		pwdCost:         pwdCost,
	}

	return auth, nil
}

func (a *LocalFileAuthenticator) Authenticate(username string, credentials []byte) (*AuthenticatedUser, error) {
	if !a.UserExists(username) {
		return nil, fmt.Errorf("user does not exist")
	}

	pwdHash, _ := a.users[username]
	err := bcrypt.CompareHashAndPassword([]byte(pwdHash), credentials)
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
	_, ok := a.users[username]
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

	username = strings.ToLower(username)

	_, ok := a.users[username]
	if ok {
		slog.Info("Update user", "username", username)
	} else {
		slog.Info("Add new user", "username", username)
	}
	a.users[username] = pwdHash
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

		a.users[strings.ToLower(s[0:p])] = s[p+1:]
	}
	return nil
}

func (a *LocalFileAuthenticator) SaveUsers() error {

	f, err := os.Create(a.userFile)
	if err != nil {
		return err
	}
	defer util.CloseOrLog(f)

	w := bufio.NewWriter(f)
	for name, hash := range a.users {
		_, err := w.WriteString(name + "\t" + hash + "\n")
		if err != nil {
			return err
		}
	}
	err = w.Flush()
	if err != nil {
		return err
	}

	slog.Info("Updated user file", "path", a.userFile)
	return nil
}
