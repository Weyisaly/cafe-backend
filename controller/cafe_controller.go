package controller

import (
	"cafe/database"
	"cafe/model"
	"cafe/utils"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const uploadDir = "./uploads"

func LoginManager(c *gin.Context) {
	type Request struct {
		Login    string `form:"login" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	var req Request
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Login and password are required"})
		return
	}

	var user model.Cafe
	if err := database.DB.Where("login = ?", req.Login).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login credentials"})
		return
	}

	access, refresh, err := utils.GenerateTokens(user.UserRole, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func UpdateMyCafe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized access",
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

	var cafe model.Cafe
	if err := tx.Where("id = ?", userID.(uint)).First(&cafe).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Cafe not found for this user",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch cafe: " + err.Error(),
			})
		}
		return
	}

	if err := processLogoUpload(c, &cafe); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Logo upload failed: " + err.Error(),
		})
		return
	}

	if name := c.PostForm("name"); name != "" {
		cafe.Name = name
	}

	if newPassword := c.PostForm("password"); newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to hash password: " + err.Error(),
			})
			return
		}
		cafe.Password = string(hashedPassword)
	}

	if phoneNumbers := c.PostFormArray("phone_numbers"); len(phoneNumbers) > 0 {
		if err := tx.Where("cafe_id = ?", cafe.ID).Delete(&model.CafePhone{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to clear existing phone numbers: " + err.Error(),
			})
			return
		}

		var newPhones []model.CafePhone
		for _, phone := range phoneNumbers {
			if phone == "" {
				continue
			}
			newPhones = append(newPhones, model.CafePhone{
				CafeID:      cafe.ID,
				PhoneNumber: phone,
			})
		}

		if len(newPhones) > 0 {
			if err := tx.Create(&newPhones).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Failed to save new phone numbers: " + err.Error(),
				})
				return
			}
		}
		cafe.PhoneNumbers = newPhones
	}

	if err := tx.Save(&cafe).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update cafe: " + err.Error(),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Transaction failed: " + err.Error(),
		})
		return
	}

	phoneNumbers := make([]string, len(cafe.PhoneNumbers))
	for i, pn := range cafe.PhoneNumbers {
		phoneNumbers[i] = pn.PhoneNumber
	}

	response := gin.H{
		"id":            cafe.ID,
		"name":          cafe.Name,
		"logo":          cafe.Logo,
		"phone_numbers": phoneNumbers,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cafe updated successfully",
		"data":    response,
	})
}

func processLogoUpload(c *gin.Context, cafe *model.Cafe) error {
	file, err := c.FormFile("logo")
	if err != nil {
		if err == http.ErrMissingFile {
			return nil
		}
		return fmt.Errorf("failed to get uploaded file: %v", err)
	}

	if file.Size > 5<<20 {
		return fmt.Errorf("file too large (max 5MB)")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	if !allowedExts[ext] {
		return fmt.Errorf("invalid file type, only JPG/JPEG/PNG allowed")
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %v", err)
	}

	newFileName := fmt.Sprintf("cafe-%d-%d%s", cafe.ID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, newFileName)

	if cafe.Logo != "" {
		if err := os.Remove(filepath.Join(uploadDir, cafe.Logo)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete old logo: %v", err)
		}
	}

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	cafe.Logo = newFileName
	return nil
}

func GetCafe(c *gin.Context) {
	var cafe []model.Cafe
	if err := database.DB.Preload("PhoneNumbers").Find(&cafe).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cafes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Fetched cafes successfully",
		"data":    cafe,
	})
}

func GetCafeByID(c *gin.Context) {
	cafeID := c.Param("id")
	if cafeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Cafe ID is required",
		})
		return
	}

	cafeIDUint, err := strconv.ParseUint(cafeID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid cafe ID format",
		})
		return
	}

	var cafe model.Cafe
	result := database.DB.Preload("PhoneNumbers").First(&cafe, uint(cafeIDUint))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Cafe not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch cafe",
			})
		}
		return
	}

	response := gin.H{
		"success": true,
		"data": gin.H{
			"id":            cafe.ID,
			"name":          cafe.Name,
			"user_role":     cafe.UserRole,
			"logo":          cafe.Logo,
			"code":          cafe.Code,
			"expiry_date":   cafe.ExpiryDate,
			"phone_numbers": cafe.PhoneNumbers,
		},
	}

	c.JSON(http.StatusOK, response)
}

func GetMyCafe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized access",
		})
		return
	}

	var cafe model.Cafe
	result := database.DB.Preload("PhoneNumbers").First(&cafe, userID.(uint))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Cafe not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch cafe",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":            cafe.ID,
			"name":          cafe.Name,
			"user_role":     cafe.UserRole,
			"logo":          cafe.Logo,
			"code":          cafe.Code,
			"expiry_date":   cafe.ExpiryDate,
			"phone_numbers": cafe.PhoneNumbers,
		},
	})
}

func RefreshTokenFunc(c *gin.Context) {
	oldRefreshToken := c.PostForm("refresh_token")
	if oldRefreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token is required"})
		return
	}

	newAccessToken, newRefreshToken, err := utils.RefreshTokens(oldRefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	})
}
