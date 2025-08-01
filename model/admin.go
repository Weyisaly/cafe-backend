package model

import (
	"gorm.io/gorm"
)

type UserRole string

const (
	Admin UserRole = "admin"
)

type User struct {
	gorm.Model
	UserCode     string   `json:"user_code"`
	Avatar       string   `json:"avatar"`
	FirstName    string   `json:"first_name"`
	LastName     string   `json:"last_name"`
	Email        string   `json:"email" gorm:"index"`
	BusinessLogo string   `json:"business_logo"`
	BusinessName string   `json:"business_name"`
	CountryID    *uint    `json:"country_id"`
	CityID       *uint    `json:"city_id"`
	Address      string   `json:"address"`
	Status       string   `json:"status"`
	Role         UserRole `json:"role"`
	PhoneNumber  string   `json:"phone_number"`
	Password     string   `json:"password"`
}
