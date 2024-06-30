package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func GetProjectHasKey(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("projectID")

		var project entity.Project
		if err := ctx.DB.Preload("KeyFile").Where("id = ?", projectID).First(&project).Error; err != nil {
			ctx.Logger.Error("Failed to get whether company has key", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get whether company has key"})
			return
		}

		if project.KeyFile != nil {
			c.JSON(http.StatusOK, gin.H{"hasKey": true, "createdAt": project.KeyFile.CreatedAt})
			return
		}

		c.JSON(http.StatusOK, gin.H{"hasKey": false, "createdAt": nil})
	}
}

func GetProjectsByUserID(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID from claims"})
			return
		}

		var user entity.User
		if err := ctx.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.Logger.Error("Failed to get user by ID", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user by ID"})
			return
		}

		var projects []entity.Project
		if err := ctx.DB.Where("company_id = ?", user.CompanyID).Find(&projects).Error; err != nil {
			ctx.Logger.Error("Failed to get projects", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get projects"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"projects": projects})
	}
}
