package controller

import (
	"cafe/database"
	"cafe/model"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func AddFood(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User ID not found in context",
		})
		return
	}

	price, err := strconv.ParseFloat(c.PostForm("price"), 64)
	if err != nil || price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid or missing price",
		})
		return
	}

	categoryID, err := strconv.ParseUint(c.PostForm("category_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid category ID format",
		})
		return
	}

	var food model.Food
	food.CafeID = userID.(uint)
	food.Price = price
	food.CategoryID = uint(categoryID)
	food.NameTm = c.PostForm("name_tm")
	food.NameRu = c.PostForm("name_ru")
	food.DescriptionTm = c.PostForm("description_tm")
	food.DescriptionRu = c.PostForm("description_ru")

	// Validate at least one name is provided
	if food.NameTm == "" && food.NameRu == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least one food name (TM, RU, or EN) is required",
		})
		return
	}

	// Validate category belongs to the user's cafe
	var category model.FoodCategory
	if err := database.DB.Where("id = ? AND cafe_id = ?", categoryID, userID.(uint)).First(&category).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid category or you don't have permission",
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

		newFileName := fmt.Sprintf("food-%d-%d%s", food.CafeID, time.Now().UnixNano(), ext)
		filePath := filepath.Join(uploadDir, newFileName)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to save image: %v", err),
			})
			return
		}

		food.Image = newFileName
	}

	if err := tx.Create(&food).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to create food: %v", err),
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
		"message": "Food added successfully",
		"data":    food,
	})
}

func BulkAddFood(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User ID not found in context"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Excel file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Unable to open Excel file"})
		return
	}
	defer file.Close()

	xl, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to parse Excel file"})
		return
	}

	rows, err := xl.GetRows("Sheet1")
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Excel must have at least one row of data"})
		return
	}

	var foods []model.Food
	for rowIndex, row := range rows[1:] {
		fmt.Printf("Row %d: %v\n", rowIndex+2, row)

		if len(row) < 4 {
			fmt.Println("⚠️ Incomplete row skipped")
			continue
		}

		price, err := strconv.ParseFloat(row[1], 64)
		if err != nil || price <= 0 {
			fmt.Println("❌ Invalid price format:", row[1])
			continue
		}

		categoryID, err := strconv.ParseUint(row[0], 10, 32)
		if err != nil {
			fmt.Println("❌ Invalid category ID:", row[0])
			continue
		}

		descriptionTm := "-"
		if len(row) > 4 && row[4] != "" {
			descriptionTm = row[4]
		}

		descriptionRu := "-"
		if len(row) > 5 && row[5] != "" {
			descriptionRu = row[5]
		}

		food := model.Food{
			CafeID:        userID.(uint),
			CategoryID:    uint(categoryID),
			Price:         price,
			NameTm:        row[2],
			NameRu:        row[3],
			DescriptionTm: descriptionTm,
			DescriptionRu: descriptionRu,
		}

		if food.NameTm == "" && food.NameRu == "" {
			fmt.Println("⚠️ Both names are empty, skipping")
			continue
		}

		foods = append(foods, food)
	}

	if len(foods) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "No valid rows found"})
		return
	}

	if err := database.DB.Create(&foods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to insert foods"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bulk food upload successful",
		"count":   len(foods),
	})
}

func UpdateFood(c *gin.Context) {
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

	var food model.Food
	if err := tx.First(&food, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Food not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to fetch food: %v", err),
			})
		}
		return
	}

	if food.CafeID != userID.(uint) {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to update this food item",
		})
		return
	}

	if nameTm := c.PostForm("name_tm"); nameTm != "" {
		food.NameTm = nameTm
	}
	if nameRu := c.PostForm("name_ru"); nameRu != "" {
		food.NameRu = nameRu
	}

	if descriptionTm := c.PostForm("description_tm"); descriptionTm != "" {
		food.DescriptionTm = descriptionTm
	}
	if descriptionRu := c.PostForm("description_ru"); descriptionRu != "" {
		food.DescriptionRu = descriptionRu
	}

	if price := c.PostForm("price"); price != "" {
		priceFloat, err := strconv.ParseFloat(price, 64)
		if err != nil || priceFloat <= 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid or negative price",
			})
			return
		}
		food.Price = priceFloat
	}
	if categoryID := c.PostForm("category_id"); categoryID != "" {
		categoryIDUint, err := strconv.ParseUint(categoryID, 10, 32)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid category ID format",
			})
			return
		}
		var category model.FoodCategory
		if err := tx.Where("id = ? AND cafe_id = ?", categoryIDUint, userID.(uint)).First(&category).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid category or you don't have permission",
			})
			return
		}
		food.CategoryID = uint(categoryIDUint)
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

		if food.Image != "" {
			if err := os.Remove(filepath.Join(uploadDir, food.Image)); err != nil && !os.IsNotExist(err) {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   fmt.Sprintf("Failed to delete old image: %v", err),
				})
				return
			}
		}

		newFileName := fmt.Sprintf("food-%d-%d%s", food.CafeID, time.Now().UnixNano(), ext)
		filePath := filepath.Join(uploadDir, newFileName)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to save new image: %v", err),
			})
			return
		}

		food.Image = newFileName
	}

	if err := tx.Save(&food).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to update food: %v", err),
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
		"message": "Food updated successfully",
		"data":    food,
	})
}

func DeleteFood(c *gin.Context) {
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

	var food model.Food
	if err := tx.First(&food, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Food not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to fetch food: %v", err),
			})
		}
		return
	}

	if food.CafeID != userID.(uint) {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to delete this food item",
		})
		return
	}

	if food.Image != "" {
		if err := os.Remove(filepath.Join(uploadDir, food.Image)); err != nil && !os.IsNotExist(err) {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to delete image: %v", err),
			})
			return
		}
	}

	if err := tx.Delete(&food).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to delete food: %v", err),
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
		"message": "Food deleted successfully",
		"data":    gin.H{"food_id": id},
	})
}

func GetFoodsByCategoryID(c *gin.Context) {
	categoryID := c.Param("category_id")
	if categoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Category ID is required",
		})
		return
	}

	categoryIDUint, err := strconv.ParseUint(categoryID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid category ID format",
		})
		return
	}

	var foods []model.Food
	if err := database.DB.Where("category_id = ?", uint(categoryIDUint)).Find(&foods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch foods: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Foods retrieved successfully",
		"data":    foods,
	})
}

func GetMyCafeFoods(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User ID not found in context",
		})
		return
	}

	searchQuery := c.Query("search")

	var foods []model.Food
	query := database.DB.Where("cafe_id = ?", userID.(uint))

	if searchQuery != "" {
		searchPattern := "%" + searchQuery + "%"
		query = query.Where("name_tm ILIKE ? OR name_ru ILIKE ?", searchPattern, searchPattern)
	}

	if err := query.Find(&foods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch foods: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Foods retrieved successfully",
		"data":    foods,
	})
}

func GetFoodByID(c *gin.Context) {
	foodID := c.Param("id")
	if foodID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Food ID is required",
		})
		return
	}

	foodIDUint, err := strconv.ParseUint(foodID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid food ID format",
		})
		return
	}

	var food model.Food
	if err := database.DB.First(&food, uint(foodIDUint)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Food not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Failed to fetch food: %v", err),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Food retrieved successfully",
		"data":    food,
	})
}
