package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func GetDatasets(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("projectID")

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

		var datasets []entity.Dataset
		if err := ctx.DB.Preload("Tables").Where("project_id = ?", projectID).Find(&datasets).Error; err != nil {
			ctx.Logger.Error("Failed to fetch datasets", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch datasets"})
			return
		}

		var response []map[string]interface{}
		for _, dataset := range datasets {
			response = append(response, map[string]interface{}{
				"id":          dataset.ID,
				"name":        dataset.Name,
				"description": dataset.Description,
				"table_count": len(dataset.Tables),
			})
		}

		c.JSON(http.StatusOK, gin.H{"datasets": response})
	}
}
