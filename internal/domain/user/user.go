package user

import (
	"strings"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Role string

const (
	RoleCustomer Role = "customer"
	RoleAdmin    Role = "admin"
)

func (r Role) Valid() bool {
	switch r {
	case RoleCustomer, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	shared.AggregateRoot
	id           uint64
	email        string
	passwordHash string
	fullName     string
	role         Role
	createdAt    time.Time
}

func NewUser(email, passwordHash, fullName string, role Role) (*User, error) {
	if email == "" {
		return nil, errs.Validation("user.email_required", "email is required")
	}
	if !strings.Contains(email, "@") {
		return nil, errs.Validation("user.email_invalid", "email must contain @")
	}
	if passwordHash == "" {
		return nil, errs.Validation("user.password_required", "password hash is required")
	}
	if role == "" {
		role = RoleCustomer
	} else if !role.Valid() {
		return nil, errs.Validation("user.role_invalid", "role is not valid")
	}

	u := &User{
		email:        email,
		passwordHash: passwordHash,
		fullName:     fullName,
		role:         role,
	}
	u.Record(UserRegistered{UserID: u.id, Email: u.email})
	return u, nil
}

func ReconstituteUser(id uint64, email, passwordHash, fullName string, role Role, createdAt time.Time) *User {
	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		fullName:     fullName,
		role:         role,
		createdAt:    createdAt,
	}
}

func (u *User) UpdateProfile(fullName string) {
	u.fullName = fullName
}

func (u *User) IsAdmin() bool {
	return u.role == RoleAdmin
}

func (u *User) AssignID(id uint64) {
	if u.id == 0 {
		u.id = id
	}
}

func (u *User) ID() uint64 {
	return u.id
}

func (u *User) Email() string {
	return u.email
}

func (u *User) PasswordHash() string {
	return u.passwordHash
}

func (u *User) FullName() string {
	return u.fullName
}

func (u *User) Role() Role {
	return u.role
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}
