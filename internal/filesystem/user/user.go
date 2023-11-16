package user

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

type User struct {
	Username     string
	UserId       uint16
	GroupId      uint16
	PasswordHash string
}

func NewUser(username, password string) *User {
	return &User{
		Username:     username,
		UserId:       0,
		GroupId:      0,
		PasswordHash: hashPassword(password),
	}
}

func (u User) GetUserString() string {
	return fmt.Sprintf("%s %d %d %s", u.Username, u.UserId, u.GroupId, u.PasswordHash)
}

func hashPassword(password string) string {
	hasher := sha512.New()
	hasher.Write([]byte(password))
	hashedPassword := hex.EncodeToString(hasher.Sum(nil))
	return hashedPassword
}
