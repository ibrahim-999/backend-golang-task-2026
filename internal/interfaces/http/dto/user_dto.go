package dto

import "github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Role     string `json:"role" binding:"omitempty,oneof=customer admin"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Role     string `json:"role" binding:"omitempty,oneof=customer admin"`
}

type UpdateUserRequest struct {
	FullName string `json:"full_name" binding:"required"`
}

type UserResponse struct {
	ID       uint64 `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

func NewUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:       u.ID(),
		Email:    u.Email(),
		FullName: u.FullName(),
		Role:     string(u.Role()),
	}
}
