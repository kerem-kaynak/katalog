package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/zap"
)

func SearchResources(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Query("project_id")
		query := c.Query("q")

		if projectID == "" || query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing project_id or search query"})
			return
		}

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

		var typeFilter string
		var actualQuery string

		switch {
		case strings.HasPrefix(query, "ds:"):
			typeFilter = "type = dataset"
			actualQuery = strings.TrimPrefix(query, "ds:")
		case strings.HasPrefix(query, "col:"):
			typeFilter = "type = column"
			actualQuery = strings.TrimPrefix(query, "col:")
		case strings.HasPrefix(query, "tab:"):
			typeFilter = "type = table"
			actualQuery = strings.TrimPrefix(query, "tab:")
		default:
			typeFilter = "type IN [dataset, column, table]"
			actualQuery = query
		}

		filter := fmt.Sprintf("project_id = %s AND %s", projectID, typeFilter)

		searchParams := &meilisearch.SearchRequest{
			Query:  actualQuery,
			Filter: filter,
		}

		searchResult, err := ctx.MeilisearchClient.Index("resources").Search(actualQuery, searchParams)
		if err != nil {
			ctx.Logger.Error("Failed to perform search", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to perform search"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"results": searchResult.Hits})
	}
}
