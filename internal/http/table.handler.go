package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func GetTables(ctx *appcontext.Context) gin.HandlerFunc {
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

		var tables []entity.Table
		if err := ctx.DB.Joins("JOIN datasets ON tables.dataset_id = datasets.id").Where("datasets.company_id = ?", user.CompanyID).Preload("Columns").Find(&tables).Error; err != nil {
			ctx.Logger.Error("Failed to fetch tables", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tables"})
			return
		}

		var response []map[string]interface{}
		for _, table := range tables {
			response = append(response, map[string]interface{}{
				"id":           table.ID,
				"name":         table.Name,
				"description":  table.Description,
				"dataset_id":   table.DatasetID,
				"column_count": len(table.Columns),
				"row_count":    table.RowCount,
			})
		}

		c.JSON(http.StatusOK, gin.H{"tables": response})
	}
}

func GetTablesByDatasetID(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		datasetID := c.Param("datasetID")

		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userHasAccess := utils.UserHasDatasetAccess(ctx, userID, uuid.MustParse(datasetID))
		if !userHasAccess {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User does not have access to this resource"})
			return
		}

		var tables []entity.Table
		if err := ctx.DB.Where("dataset_id = ?", datasetID).Preload("Columns").Find(&tables).Error; err != nil {
			ctx.Logger.Error("Failed to get tables", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tables"})
			return
		}

		var response []map[string]interface{}
		for _, table := range tables {
			response = append(response, map[string]interface{}{
				"id":           table.ID,
				"name":         table.Name,
				"description":  table.Description,
				"dataset_id":   table.DatasetID,
				"column_count": len(table.Columns),
				"row_count":    table.RowCount,
			})
		}

		c.JSON(http.StatusOK, gin.H{"tables": response})
	}
}
