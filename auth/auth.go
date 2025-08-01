package auth

import (
	"cafe/database"
	"cafe/model"
	"cafe/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

func Login(c *gin.Context) {
	type Request struct {
		PhoneNumber string `form:"phone_number" binding:"required"`
		Password    string `form:"password" binding:"required"`
	}

	var req Request
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Telefon belgisi we açar sözi hökmany"})
		return
	}

	var user model.User
	if err := database.DB.Where("phone_number = ?", req.PhoneNumber).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ulanyjy tapylmady"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Nädogry açar söz"})
		return
	}

	access, refresh, err := utils.GenerateTokens(string(user.Role), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token döretmek şowsuz boldy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}
