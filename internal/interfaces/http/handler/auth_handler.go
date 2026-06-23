package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/dto"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/httpx"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Auth struct {
	register *command.RegisterUser
	login    *command.LoginUser
}

func NewAuth(register *command.RegisterUser, login *command.LoginUser) *Auth {
	return &Auth{register: register, login: login}
}

func (h *Auth) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("request.invalid", err.Error()))
		return
	}

	role := req.Role
	if role == "" {
		role = string(user.RoleCustomer)
	}

	u, err := h.register.Handle(c.Request.Context(), command.RegisterUserInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Role:     role,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.Created(c, dto.NewUserResponse(u))
}

func (h *Auth) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("request.invalid", err.Error()))
		return
	}

	token, u, err := h.login.Handle(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dto.LoginResponse{Token: token, User: dto.NewUserResponse(u)})
}
