package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func GetColumns(ctx *appcontext.Context) gin.HandlerFunc {
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

		var columns []entity.Column
		if err := ctx.DB.Joins("JOIN tables ON columns.table_id = tables.id").Joins("JOIN datasets ON tables.dataset_id = datasets.id").Where("datasets.company_id = ?", user.CompanyID).Find(&columns).Error; err != nil {
			ctx.Logger.Error("Failed to fetch columns", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch columns"})
			return
		}

		var response []map[string]interface{}
		for _, column := range columns {
			response = append(response, map[string]interface{}{
				"id":          column.ID,
				"name":        column.Name,
				"description": column.Description,
				"table_id":    column.TableID,
				"type":        column.Type,
			})
		}

		c.JSON(http.StatusOK, gin.H{"columns": response})
	}
}
