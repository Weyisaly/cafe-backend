package controller

import (
	"cafe/database"
	"cafe/model"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func AddCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User ID not found in context",
		})
		return
	}

	var category model.FoodCategory
	category.CafeId = userID.(uint)
	category.NameTM = c.PostForm("name_tm")
	category.NameRU = c.PostForm("name_ru")
	category.NameEN = c.PostForm("name_en")

	// Validate at least one name is provided
	if category.NameTM == "" && category.NameRU == "" && category.NameEN == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least one category name (TM, RU, or EN) is required",
		})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Unexpected error occurred",
			})
		}
	}()

	file, err := c.FormFile("image")
	if err == nil {
		if file.Size > 5<<20 { // 5MB limit
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Image size exceeds 5MB limit",
			})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true}
		if !allowedExts[ext] {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid file type, only JPG/JPEG/PNG allowed",
			})
			return
		}

		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to create upload directory: %v", err),
			})
			return
		}

		newFileName := fmt.Sprintf("category-%d-%d%s", category.CafeId, time.Now().UnixNano(), ext)
		filePath := filepath.Join(uploadDir, newFileName)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to save image: %v", err),
			})
			return
		}

		category.Image = newFileName
	}

	if err := tx.Create(&category).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to create category: %v", err),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Transaction failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category added successfully",
		"data":    category,
	})
}

// UpdateCategory handles updating an existing food category.
func UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User ID not found in context",
		})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Unexpected error occurred",
			})
		}
	}()

	var category model.FoodCategory
	if err := tx.First(&category, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Category not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to fetch category: %v", err),
			})
		}
		return
	}

	if category.CafeId != userID.(uint) {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to update this category",
		})
		return
	}

	if nameTM := c.PostForm("name_tm"); nameTM != "" {
		category.NameTM = nameTM
	}
	if nameRU := c.PostForm("name_ru"); nameRU != "" {
		category.NameRU = nameRU
	}
	if nameEN := c.PostForm("name_en"); nameEN != "" {
		category.NameEN = nameEN
	}

	file, err := c.FormFile("image")
	if err == nil {
		if file.Size > 5<<20 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Image size exceeds 5MB limit",
			})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true}
		if !allowedExts[ext] {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid file type, only JPG/JPEG/PNG allowed",
			})
			return
		}

		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to create upload directory: %v", err),
			})
			return
		}

		if category.Image != "" {
			if err := os.Remove(filepath.Join(uploadDir, category.Image)); err != nil && !os.IsNotExist(err) {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   fmt.Sprintf("Failed to delete old image: %v", err),
				})
				return
			}
		}

		newFileName := fmt.Sprintf("category-%d-%d%s", category.CafeId, time.Now().UnixNano(), ext)
		filePath := filepath.Join(uploadDir, newFileName)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to save new image: %v", err),
			})
			return
		}

		category.Image = newFileName
	}

	if err := tx.Save(&category).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to update category: %v", err),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Transaction failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category updated successfully",
		"data":    category,
	})
}

// DeleteCategory handles deleting a food category and its associated image.
func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User ID not found in context",
		})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Unexpected error occurred",
			})
		}
	}()

	var category model.FoodCategory
	if err := tx.First(&category, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Category not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to fetch category: %v", err),
			})
		}
		return
	}

	if category.CafeId != userID.(uint) {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to delete this category",
		})
		return
	}

	if category.Image != "" {
		if err := os.Remove(filepath.Join(uploadDir, category.Image)); err != nil && !os.IsNotExist(err) {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to delete image: %v", err),
			})
			return
		}
	}

	if err := tx.Delete(&category).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to delete category: %v", err),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Transaction failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category deleted successfully",
		"data":    gin.H{"category_id": id},
	})
}

// GetMyCategories retrieves categories for the authenticated user's cafe.
func GetMyCategories(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User ID not found in context",
		})
		return
	}

	var categories []model.FoodCategory
	if err := database.DB.Where("cafe_id = ?", userID.(uint)).Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to retrieve categories: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Categories retrieved successfully",
		"data":    categories,
	})
}

// GetCategoriesByCafeID retrieves categories for a specific cafe.
func GetCategoriesByCafeID(c *gin.Context) {
	cafeIDStr := c.Param("cafe_id")
	cafeID, err := strconv.Atoi(cafeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid cafe_id",
		})
		return
	}

	var categories []model.FoodCategory
	if err := database.DB.Where("cafe_id = ?", cafeID).Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to retrieve categories: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Categories retrieved successfully",
		"data":    categories,
	})
}

// GetCafeCategoriesWithFoods retrieves categories and their associated foods for a cafe.
func GetCafeCategoriesWithFoods(c *gin.Context) {
	cafeID := c.Query("cafe_id")
	if cafeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "cafe_id parameter is required",
		})
		return
	}

	cafeIDUint, err := strconv.ParseUint(cafeID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid cafe_id format",
		})
		return
	}

	type Result struct {
		model.FoodCategory
		Foods []model.Food `gorm:"foreignKey:CategoryID" json:"foods"`
	}

	var result []Result
	err = database.DB.
		Model(&model.FoodCategory{}).
		Where("cafe_id = ?", uint(cafeIDUint)).
		Preload("Foods", func(db *gorm.DB) *gorm.DB {
			return db.Where("cafe_id = ?", uint(cafeIDUint))
		}).
		Find(&result).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch data: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Categories and foods retrieved successfully",
		"data":    result,
	})
}
