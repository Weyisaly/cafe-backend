package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func CafeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		role, err := ExtractRoleFromToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if role != "cafe" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Cafe access required"})
			c.Abort()
			return
		}

		userID, err := ExtractIDFromToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token ID"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)

		c.Next()
	}
}

func ExtractRoleFromToken(authHeader string) (string, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("invalid token format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	role, ok := claims["user_role"].(string)
	if !ok {
		return "", errors.New("role not found in token")
	}

	return role, nil
}

func ExtractIDFromToken(authHeader string) (uint, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, errors.New("invalid token format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}

	idFloat, ok := claims["id"].(float64)
	if !ok {
		return 0, errors.New("id not found or invalid type")
	}

	return uint(idFloat), nil
}
