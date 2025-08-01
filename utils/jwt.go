package utils

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

func GenerateTokens(userRole string, userID uint) (string, string, error) {
	secretKey := os.Getenv("enweyos")
	if secretKey == "" {
		secretKey = "enweyos"
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_role": userRole,
		"id":        userID,
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	})
	access, err := accessToken.SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_role": userRole,
		"id":        userID,
		"exp":       time.Now().Add(12 * time.Hour).Unix(),
	})
	refresh, err := refreshToken.SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func ValidateToken(tokenString string) (map[string]interface{}, error) {
	secretKey := os.Getenv("enweyos")
	if secretKey == "" {
		secretKey = "enweyos"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				return nil, errors.New("token has expired")
			}
		} else {
			return nil, errors.New("invalid or missing expiration claim")
		}

		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func RefreshTokens(oldRefreshToken string) (string, string, error) {
	secretKey := os.Getenv("enweyos")
	if secretKey == "" {
		secretKey = "enweyos"
	}

	token, err := jwt.Parse(oldRefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return "", "", fmt.Errorf("error parsing refresh token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				return "", "", errors.New("refresh token has expired")
			}
		} else {
			return "", "", errors.New("invalid or missing expiration claim")
		}

		userRole, _ := claims["user_role"].(string)
		userID, _ := claims["id"].(float64)

		// Generate new access and refresh tokens
		newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_role": userRole,
			"id":        uint(userID),
			"exp":       time.Now().Add(15 * time.Minute).Unix(),
		})
		newAccess, err := newAccessToken.SignedString([]byte(secretKey))
		if err != nil {
			return "", "", err
		}

		newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_role": userRole,
			"id":        uint(userID),
			"exp":       time.Now().Add(12 * time.Hour).Unix(),
		})
		newRefresh, err := newRefreshToken.SignedString([]byte(secretKey))
		if err != nil {
			return "", "", err
		}

		return newAccess, newRefresh, nil
	}

	return "", "", errors.New("invalid refresh token")
}
