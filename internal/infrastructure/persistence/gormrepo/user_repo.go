package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type userRepo struct {
	db *gorm.DB
}

func newUserRepo(db *gorm.DB) *userRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	m := userToModel(u)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	u.AssignID(m.ID)
	return nil
}

func (r *userRepo) Update(ctx context.Context, u *user.User) error {
	m := userToModel(u)
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *userRepo) FindByID(ctx context.Context, id uint64) (*user.User, error) {
	var m UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("user.not_found", "user not found")
		}
		return nil, err
	}
	return userToDomain(m), nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var m UserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("user.not_found", "user not found")
		}
		return nil, err
	}
	return userToDomain(m), nil
}

func userToModel(u *user.User) UserModel {
	return UserModel{
		ID:           u.ID(),
		Email:        u.Email(),
		PasswordHash: u.PasswordHash(),
		FullName:     u.FullName(),
		Role:         string(u.Role()),
	}
}

func userToDomain(m UserModel) *user.User {
	return user.ReconstituteUser(m.ID, m.Email, m.PasswordHash, m.FullName, user.Role(m.Role), m.CreatedAt)
}

var _ ports.UserRepository = (*userRepo)(nil)
