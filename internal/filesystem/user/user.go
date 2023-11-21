package user

import (
	"crypto/sha512"
	"encoding/hex"
	"file-system/internal/errs"
	"fmt"
	"strconv"
	"strings"
)

type User struct {
	Username     string
	UserId       uint16
	PasswordHash string
}

func NewUser(username string, userId uint16, password string) *User {
	return &User{
		Username:     username,
		UserId:       userId,
		PasswordHash: hashPassword(password),
	}
}

func ReadUserFromString(str, password string) (*User, error) {
	parts := strings.Fields(str)

	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid input format")
	}

	userId, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("error parsing UserId: %v", err)
	}

	u := &User{
		Username:     parts[0],
		UserId:       uint16(userId),
		PasswordHash: parts[2],
	}

	if hashPassword(password) != u.PasswordHash {
		return nil, fmt.Errorf("%w - %s", errs.ErrIncorrectPassword, password)
	}

	return u, nil
}

func GetUserIdFromString(str string) (uint16, error) {
	parts := strings.Fields(str)

	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid input format")
	}

	userId, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, fmt.Errorf("error parsing UserId: %v", err)
	}

	return uint16(userId), nil
}

func (u User) GetUserString() string {
	return fmt.Sprintf("%s %d %s", u.Username, u.UserId, u.PasswordHash)
}

func hashPassword(password string) string {
	hasher := sha512.New()
	hasher.Write([]byte(password))
	hashedPassword := hex.EncodeToString(hasher.Sum(nil))
	return hashedPassword
}
