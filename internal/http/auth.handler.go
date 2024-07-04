package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/services"
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
			Sub     string `json:"sub"`
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}{}

		if err := json.Unmarshal(body, &user); err != nil {
			ctx.Logger.Error("Failed to unmarshal user info", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal user info"})
			return
		}

		var dbUser entity.User
		if err := ctx.DB.Where("email = ?", user.Email).First(&dbUser).Error; err != nil {
			dbUser = entity.User{
				Email:          user.Email,
				Name:           user.Name,
				ProfilePicture: user.Picture,
				Role:           "Admin",
			}
			if err := ctx.DB.Create(&dbUser).Error; err != nil {
				ctx.Logger.Error("Failed to create user", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
		} else {
			if dbUser.Status != "active" {
				updates := map[string]interface{}{
					"status":          "active",
					"name":            user.Name,
					"profile_picture": user.Picture,
				}
				if err := ctx.DB.Model(&dbUser).Updates(updates).Error; err != nil {
					ctx.Logger.Error("Failed to update user details", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user details"})
					return
				}
			}
		}

		tokenString, err := utils.GenerateJWT(dbUser.ID.String())
		if err != nil {
			ctx.Logger.Error("Failed to generate JWT token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token"})
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "http://localhost:3000/company/create?token="+tokenString)
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

type UserWithCompany struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	ProfilePicture string    `json:"profile_picture"`
	CompanyID      uuid.UUID `json:"company_id"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	Company        string    `json:"company"`
}

func GetUserInfoWithCompany(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var user entity.User
		if err := ctx.DB.Preload("Company").First(&user, "id = ?", userID).Error; err != nil {
			ctx.Logger.Error("Failed to find user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
			return
		}

		userWithCompany := UserWithCompany{
			ID:             user.ID,
			Name:           user.Name,
			Email:          user.Email,
			ProfilePicture: user.ProfilePicture,
			CompanyID:      *user.CompanyID,
			Role:           user.Role,
			Status:         user.Status,
			Company:        user.Company.Name,
		}

		c.JSON(http.StatusOK, gin.H{"user": userWithCompany})
	}
}

func InviteUser(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		type InviteUserRequest struct {
			Email string `json:"email"`
		}

		var request InviteUserRequest
		if err := c.BindJSON(&request); err != nil {
			ctx.Logger.Error("Failed to bind request", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to bind request"})
			return
		}

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

		var existingUser entity.User
		if err := ctx.DB.First(&existingUser, "email = ?", request.Email).Error; err != nil {
			dbUser := entity.User{
				Email:     request.Email,
				Status:    "pending",
				CompanyID: user.CompanyID,
				Role:      "Member",
			}
			if err := ctx.DB.Create(&dbUser).Error; err != nil {
				ctx.Logger.Error("Failed to create user", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
		} else {
			updates := map[string]interface{}{
				"company_id": user.CompanyID,
				"role":       "Member",
			}

			if err := ctx.DB.Model(&existingUser).Updates(updates).Error; err != nil {
				ctx.Logger.Error("Failed to update user details", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user details"})
				return
			}
		}

		loginURL := fmt.Sprintf("%s/login", os.Getenv("FRONTEND_HOST"))
		if err := services.SendInvitationEmail(request.Email, user.Name, loginURL); err != nil {
			ctx.Logger.Error("Failed to send email", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User successfully invited!"})
	}
}
