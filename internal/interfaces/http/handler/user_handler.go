package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/query"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/dto"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/httpx"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type User struct {
	register *command.RegisterUser
	update   *command.UpdateUser
	get      *query.GetUser
}

func NewUser(register *command.RegisterUser, update *command.UpdateUser, get *query.GetUser) *User {
	return &User{register: register, update: update, get: get}
}

func (h *User) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("request.invalid", err.Error()))
		return
	}

	u, err := h.register.Handle(c.Request.Context(), command.RegisterUserInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Role:     req.Role,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.Created(c, dto.NewUserResponse(u))
}

func (h *User) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		httpx.Error(c, err)
		return
	}

	u, err := h.get.Handle(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dto.NewUserResponse(u))
}

func (h *User) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		httpx.Error(c, err)
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("request.invalid", err.Error()))
		return
	}

	u, err := h.update.Handle(c.Request.Context(), id, req.FullName)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dto.NewUserResponse(u))
}

func parseID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errs.Validation("request.invalid_id", "invalid resource id")
	}
	return id, nil
}
