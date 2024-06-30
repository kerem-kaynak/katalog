package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"go.uber.org/zap"
)

func GetColumnsByTableID(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("tableID")

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
