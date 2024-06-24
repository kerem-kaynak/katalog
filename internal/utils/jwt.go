package utils

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var jwtKey = []byte("your_secret_key")

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func GetUserIDFromClaims(c *gin.Context) (uuid.UUID, error) {
	claims, exists := c.Get("claims")
	if !exists {
		return uuid.Nil, errors.New("claims not found in context")
	}

	registeredClaims, ok := claims.(*Claims)
	if !ok {
		return uuid.Nil, errors.New("claims are not of type *Claims")
	}

	uuidUserID, err := uuid.Parse(registeredClaims.UserID)
	if err != nil {
		return uuid.Nil, errors.New("invalid user ID format")
	}

	return uuidUserID, nil
}
