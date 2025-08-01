package model

import "gorm.io/gorm"

type Food struct {
	gorm.Model
	CafeID        uint    `json:"cafe_id"`
	CategoryID    uint    `json:"category_id"`
	Image         string  `json:"image"`
	Price         float64 `json:"price"`
	NameTm        string  `json:"name_tm"`
	NameRu        string  `json:"name_ru"`
	DescriptionTm string  `json:"description_tm"`
	DescriptionRu string  `json:"description_ru"`
}
