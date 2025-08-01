package model

import (
	"gorm.io/gorm"
	"time"
)

type Cafe struct {
	gorm.Model
	Login        string      `json:"login"`
	Password     string      `json:"password"`
	Name         string      `json:"name"`
	UserRole     string      `json:"user_role"`
	Logo         string      `json:"logo"`
	Code         string      `json:"code"`
	PhoneNumbers []CafePhone `json:"phone_numbers" gorm:"foreignKey:CafeID"`
	ExpiryDate   time.Time   `json:"expiry_date"`
}

type CafePhone struct {
	gorm.Model
	CafeID      uint   `json:"cafe_id"`
	PhoneNumber string `json:"phone_number"`
}
