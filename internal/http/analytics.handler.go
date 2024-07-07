package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func GetDashboardStatistics(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("projectID")

		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userHasAccess := utils.UserHasProjectAccess(ctx, userID, uuid.MustParse(projectID))
		if !userHasAccess {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User does not have access to this resource"})
			return
		}

		now := time.Now()
		currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		pastMonthStart := currentMonthStart.AddDate(0, -1, 0)

		var totalDatasetCount int64
		ctx.DB.Model(&entity.Dataset{}).Where("project_id = ?", projectID).Count(&totalDatasetCount)

		var totalTableCount int64
		ctx.DB.Model(&entity.Table{}).Joins("JOIN datasets ON datasets.id = tables.dataset_id").Where("datasets.project_id = ? AND tables.deleted_at IS NULL", projectID).Count(&totalTableCount)

		var totalColumnCount int64
		ctx.DB.Model(&entity.Column{}).Joins("JOIN tables ON tables.id = columns.table_id").Joins("JOIN datasets ON datasets.id = tables.dataset_id").Where("datasets.project_id = ? AND columns.deleted_at IS NULL AND tables.deleted_at IS NULL", projectID).Count(&totalColumnCount)

		var totalRowCount int64
		ctx.DB.Model(&entity.Table{}).Select("SUM(row_count)").Joins("JOIN datasets ON datasets.id = tables.dataset_id").Where("datasets.project_id = ? AND tables.deleted_at IS NULL", projectID).Scan(&totalRowCount)

		var pastMonthDatasetCount int64
		ctx.DB.Model(&entity.Dataset{}).Where("project_id = ? AND created_at < ?", projectID, currentMonthStart).Count(&pastMonthDatasetCount)

		var pastMonthTableCount int64
		ctx.DB.Model(&entity.Table{}).Joins("JOIN datasets ON datasets.id = tables.dataset_id").Where("datasets.project_id = ? AND tables.created_at < ? AND tables.deleted_at IS NULL", projectID, currentMonthStart).Count(&pastMonthTableCount)

		var pastMonthColumnCount int64
		ctx.DB.Model(&entity.Column{}).Joins("JOIN tables ON tables.id = columns.table_id").Joins("JOIN datasets ON datasets.id = tables.dataset_id").Where("datasets.project_id = ? AND columns.created_at < ? AND columns.deleted_at IS NULL AND tables.deleted_at IS NULL", projectID, currentMonthStart).Count(&pastMonthColumnCount)

		var pastMonthTotalRowCount int64
		ctx.DB.Model(&entity.Table{}).Select("SUM(row_count)").Joins("JOIN datasets ON datasets.id = tables.dataset_id").Where("datasets.project_id = ? AND tables.created_at < ? AND tables.deleted_at IS NULL", projectID, currentMonthStart).Scan(&pastMonthTotalRowCount)

		var tableCountsRaw []struct {
			DatasetName string
			Count       int64
		}
		ctx.DB.Table("datasets").
			Select("datasets.name as dataset_name, COUNT(tables.id) as count").
			Joins("LEFT JOIN tables ON datasets.id = tables.dataset_id AND tables.deleted_at IS NULL").
			Where("datasets.project_id = ? AND datasets.deleted_at IS NULL", projectID).
			Group("datasets.name").
			Scan(&tableCountsRaw)

		tableCountsResponse := struct {
			DatasetNames []string `json:"datasetNames"`
			TableCounts  []int64  `json:"tableCounts"`
		}{}

		for _, item := range tableCountsRaw {
			tableCountsResponse.DatasetNames = append(tableCountsResponse.DatasetNames, item.DatasetName)
			tableCountsResponse.TableCounts = append(tableCountsResponse.TableCounts, item.Count)
		}

		var tableSizeMetricRaw []struct {
			DatasetName     string
			AvgNumberOfRows float64
			NumberOfTables  int64
		}
		ctx.DB.Table("datasets").
			Select("datasets.name as dataset_name, AVG(tables.row_count) as avg_number_of_rows, COUNT(tables.id) as number_of_tables").
			Joins("LEFT JOIN tables ON datasets.id = tables.dataset_id AND tables.deleted_at IS NULL").
			Where("datasets.project_id = ? AND datasets.deleted_at IS NULL", projectID).
			Group("datasets.name").
			Scan(&tableSizeMetricRaw)

		tableSizeMetricResponse := struct {
			DatasetNames []string  `json:"datasetNames"`
			AvgRowCounts []float64 `json:"avgRowCounts"`
			TableCounts  []int64   `json:"tableCounts"`
		}{}

		for _, item := range tableSizeMetricRaw {
			tableSizeMetricResponse.DatasetNames = append(tableSizeMetricResponse.DatasetNames, item.DatasetName)
			tableSizeMetricResponse.AvgRowCounts = append(tableSizeMetricResponse.AvgRowCounts, item.AvgNumberOfRows)
			tableSizeMetricResponse.TableCounts = append(tableSizeMetricResponse.TableCounts, item.NumberOfTables)
		}

		var columnTypeDistributionRaw []struct {
			Type  string
			Count int64
		}
		ctx.DB.Table("columns").
			Select("columns.type, COUNT(*) as count").
			Joins("JOIN tables ON tables.id = columns.table_id").
			Joins("JOIN datasets ON datasets.id = tables.dataset_id").
			Where("datasets.project_id = ? AND columns.deleted_at IS NULL AND tables.deleted_at IS NULL", projectID).
			Group("columns.type").
			Scan(&columnTypeDistributionRaw)

		columnTypeDistributionResponse := []struct {
			ID    int    `json:"id"`
			Label string `json:"label"`
			Value int64  `json:"value"`
		}{}

		for i, item := range columnTypeDistributionRaw {
			columnTypeDistributionResponse = append(columnTypeDistributionResponse, struct {
				ID    int    `json:"id"`
				Label string `json:"label"`
				Value int64  `json:"value"`
			}{
				ID:    i + 1,
				Label: item.Type,
				Value: item.Count,
			})
		}

		var currentMonthSyncCount int64
		ctx.DB.Model(&entity.Sync{}).Where("project_id = ? AND created_at >= ?", projectID, currentMonthStart).Count(&currentMonthSyncCount)

		var pastMonthSyncCount int64
		ctx.DB.Model(&entity.Sync{}).Where("project_id = ? AND created_at >= ? AND created_at < ?", projectID, pastMonthStart, currentMonthStart).Count(&pastMonthSyncCount)

		var currentMonthChangeCountsRaw []struct {
			ChangeType string
			Count      int64
		}
		ctx.DB.Model(&entity.Changelog{}).
			Select("changelogs.change_type, COUNT(*) as count").
			Joins("JOIN syncs ON syncs.id = changelogs.sync_id").
			Where("syncs.project_id = ? AND changelogs.created_at >= ? AND changelogs.deleted_at IS NULL", projectID, currentMonthStart).
			Group("changelogs.change_type").
			Scan(&currentMonthChangeCountsRaw)

		currentMonthChangeCountsResponse := struct {
			Insert int64 `json:"insert"`
			Update int64 `json:"update"`
			Delete int64 `json:"delete"`
		}{}

		for _, item := range currentMonthChangeCountsRaw {
			switch item.ChangeType {
			case "insert":
				currentMonthChangeCountsResponse.Insert = item.Count
			case "update":
				currentMonthChangeCountsResponse.Update = item.Count
			case "delete":
				currentMonthChangeCountsResponse.Delete = item.Count
			}
		}

		// Prepare the response structure
		response := gin.H{
			"totalDatasetCount":        totalDatasetCount,
			"totalTableCount":          totalTableCount,
			"totalColumnCount":         totalColumnCount,
			"totalRowCount":            totalRowCount,
			"pastMonthDatasetCount":    pastMonthDatasetCount,
			"pastMonthTableCount":      pastMonthTableCount,
			"pastMonthColumnCount":     pastMonthColumnCount,
			"pastMonthTotalRowCount":   pastMonthTotalRowCount,
			"tableCounts":              tableCountsResponse,
			"tableSizeMetric":          tableSizeMetricResponse,
			"columnTypeDistribution":   columnTypeDistributionResponse,
			"currentMonthSyncCount":    currentMonthSyncCount,
			"pastMonthSyncCount":       pastMonthSyncCount,
			"currentMonthChangeCounts": currentMonthChangeCountsResponse,
		}

		// Send the response
		c.JSON(http.StatusOK, response)
	}
}
