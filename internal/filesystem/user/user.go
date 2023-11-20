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
	GroupId      uint16
	PasswordHash string
}

func NewUser(username string, userId uint16, groupId uint16, password string) *User {
	return &User{
		Username:     username,
		UserId:       userId,
		GroupId:      groupId,
		PasswordHash: hashPassword(password),
	}
}

func ReadUserFromString(str, password string) (*User, error) {
	parts := strings.Fields(str)

	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid input format")
	}

	userId, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("error parsing UserId: %v", err)
	}

	groupId, err := strconv.ParseUint(parts[2], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("error parsing GroupId: %v", err)
	}

	u := &User{
		Username:     parts[0],
		UserId:       uint16(userId),
		GroupId:      uint16(groupId),
		PasswordHash: parts[3],
	}

	if hashPassword(password) != u.PasswordHash {
		return nil, fmt.Errorf("%w - %s", errs.ErrIncorrectPassword, password)
	}

	return u, nil
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
