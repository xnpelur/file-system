package usermanager

import "file-system/internal/filesystem/user"

type UserManager struct {
	Current *user.User
	users   map[uint16]string
	nextId  uint16
}

func NewUserManager() *UserManager {
	return &UserManager{
		users: make(map[uint16]string),
	}
}

func (um *UserManager) CreateNewUser(username, password string) *user.User {
	newUser := user.NewUser(username, um.nextId, password)
	um.users[um.nextId] = username
	um.nextId++
	return newUser
}

func (um *UserManager) LoadUsers(users map[uint16]string) {
	um.users = users
	for key := range users {
		if key+1 > um.nextId {
			um.nextId = key + 1
		}
	}
}

func (um *UserManager) GetUsername(userId uint16) string {
	return um.users[userId]
}

func (um *UserManager) DeleteUser(userId uint16) {
	delete(um.users, userId)
}
