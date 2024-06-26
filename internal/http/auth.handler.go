package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func Login(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := ctx.OAuth2Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func Callback(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		token, err := ctx.OAuth2Config.Exchange(context.Background(), code)
		if err != nil {
			ctx.Logger.Error("Failed to exchange token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
			return
		}

		client := ctx.OAuth2Config.Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			ctx.Logger.Error("Failed to get user info", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			ctx.Logger.Error("Failed to read user info response body", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user info response body"})
			return
		}

		user := struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}{}

		if err := json.Unmarshal(body, &user); err != nil {
			ctx.Logger.Error("Failed to unmarshal user info", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal user info"})
			return
		}

		var dbUser entity.User
		if err := ctx.DB.Where("email = ?", user.Email).First(&dbUser).Error; err != nil {
			companyID := uuid.MustParse("5366a36e-9bcd-46a8-b84a-4935842a1d91")
			dbUser = entity.User{
				Email:     user.Email,
				Name:      user.Name,
				CompanyID: &companyID,
			}
			if err := ctx.DB.Create(&dbUser).Error; err != nil {
				ctx.Logger.Error("Failed to create user", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
		}

		tokenString, err := utils.GenerateJWT(dbUser.ID.String())
		if err != nil {
			ctx.Logger.Error("Failed to generate JWT token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token"})
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "http://localhost:3000/login?token="+tokenString)
	}
}

func Logout(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie("token", "", -1, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
	}
}

func GetUserInfo(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var user entity.User
		if err := ctx.DB.First(&user, "id = ?", userID).Error; err != nil {
			ctx.Logger.Error("Failed to find user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
