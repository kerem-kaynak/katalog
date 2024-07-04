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

func GetColumnsByTableID(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("tableID")

		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userHasAccess := utils.UserHasTableAccess(ctx, userID, uuid.MustParse(tableID))
		if !userHasAccess {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User does not have access to this resource"})
			return
		}

		var columns []entity.Column
		if err := ctx.DB.Where("table_id = ?", tableID).Find(&columns).Error; err != nil {
			ctx.Logger.Error("Failed to get columns", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get columns"})
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
