package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func GetCompanyMembers(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID from claims"})
			return
		}

		var user entity.User
		if err := ctx.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.Logger.Error("Failed to get user from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user from database"})
			return
		}

		var companyMembers []entity.User
		if err := ctx.DB.Where("company_id = ?", user.CompanyID).Find(&companyMembers).Error; err != nil {
			ctx.Logger.Error("Failed to get team members from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get team members from database"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"members": companyMembers})
	}
}

func GetCompanyHasKey(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID from claims"})
			return
		}

		var user entity.User
		if err := ctx.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.Logger.Error("Failed to get user from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user from database"})
			return
		}

		var company entity.Company
		if err := ctx.DB.Preload("KeyFile").Where("id = ?", user.CompanyID).First(&company).Error; err != nil {
			ctx.Logger.Error("Failed to get whether company has key", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get whether company has key"})
			return
		}

		if company.KeyFile != nil {
			c.JSON(http.StatusOK, gin.H{"hasKey": true})
			return
		}

		c.JSON(http.StatusOK, gin.H{"hasKey": false})
	}
}
