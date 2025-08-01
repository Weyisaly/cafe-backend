package model

import "gorm.io/gorm"

type FoodCategory struct {
	gorm.Model
	NameTM string `json:"name_tm"`
	NameRU string `json:"name_ru"`
	NameEN string `json:"name_en"`
	Image  string `json:"image"`
	CafeId uint   `json:"cafe_id"`
}
