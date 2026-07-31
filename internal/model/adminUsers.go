package model

import (
	"go_server/config"

	"golang.org/x/crypto/bcrypt"
)

type AdminUsers struct {
	BaseModel
	Username string `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password string `gorm:"size:255;not null" json:"-"`
	Email    string `gorm:"size:100" json:"email"`
	Phone    string `gorm:"size:20" json:"phone"`
	Status   int    `gorm:"default:1" json:"status"`
	RoleID   uint   `gorm:"index" json:"role_id"`
}

func (AdminUsers) TableName() string {
	return config.AppConfig.Database.Prefix + "admin_users"
}

func (u *AdminUsers) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

func (u *AdminUsers) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
