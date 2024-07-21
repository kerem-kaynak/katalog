package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gorm.io/gorm/clause"
)

type ServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func FetchSchema(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectIDString := c.Param("projectID")
		projectID := uuid.MustParse(projectIDString)

		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userHasAccess := utils.UserHasProjectAccess(ctx, userID, projectID)
		if !userHasAccess {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User does not have access to this resource"})
			return
		}

		var user entity.User
		if err := ctx.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.Logger.Error("Failed to get user from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user from database"})
			return
		}

		var keyFile entity.KeyFile
		if err := ctx.DB.Where("project_id = ?", projectID).First(&keyFile).Error; err != nil {
			ctx.Logger.Error("Failed to get key file for user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get key file for user"})
			return
		}

		rc, err := ctx.GCSClient.Bucket(ctx.GCSBucketName).Object(projectIDString + "/sa_key").NewReader(context.Background())
		if err != nil {
			ctx.Logger.Error("Failed to fetch key file from GCS", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch key file from GCS"})
			return
		}
		defer rc.Close()

		keyFileBytes, err := io.ReadAll(rc)
		if err != nil {
			ctx.Logger.Error("Failed to read key file from GCS", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read key file from GCS"})
			return
		}

		var key ServiceAccountKey
		if err := json.Unmarshal(keyFileBytes, &key); err != nil {
			ctx.Logger.Error("Failed to unmarshal key file", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal key file"})
			return
		}

		conf, err := google.JWTConfigFromJSON(keyFileBytes, bigquery.Scope)
		if err != nil {
			ctx.Logger.Error("Failed to parse key file", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse key file"})
			return
		}

		client, err := bigquery.NewClient(context.Background(), key.ProjectID, option.WithTokenSource(conf.TokenSource(context.Background())))
		if err != nil {
			ctx.Logger.Error("Failed to create BigQuery client", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create BigQuery client"})
			return
		}
		defer client.Close()

		oldState, err := utils.FetchCurrentState(ctx, projectID)
		if err != nil {
			ctx.Logger.Error("Failed to fetch old state for changelog", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch old state for changelog"})
			return
		}

		it := client.Datasets(context.Background())

		tx := ctx.DB.Begin()
		if err := tx.Error; err != nil {
			ctx.Logger.Error("Failed to begin transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
			return
		}

		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		if err := tx.Model(&entity.Dataset{}).Where("project_id = ?", projectID).Update("to_delete", true).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to mark datasets potentially for delete", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark datasets potentially for delete"})
			return
		}
		if err := tx.Model(&entity.Table{}).Where("dataset_id IN (SELECT id FROM datasets WHERE project_id = ?)", projectID).Update("to_delete", true).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to mark tables potentially for delete", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark tables potentially for delete"})
			return
		}
		if err := tx.Model(&entity.Column{}).Where("table_id IN (SELECT id FROM tables WHERE dataset_id IN (SELECT id FROM datasets WHERE project_id = ?))", projectID).Update("to_delete", true).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to mark columns potentially for delete", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark columns potentially for delete"})
			return
		}

		var documentsToIndex []map[string]interface{}

		for {
			ds, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				tx.Rollback()
				ctx.Logger.Error("Failed to fetch datasets", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch datasets"})
				return
			}

			meta, err := ds.Metadata(context.Background())
			if err != nil {
				tx.Rollback()
				ctx.Logger.Error("Failed to fetch dataset metadata", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dataset metadata"})
				return
			}

			dataset := entity.Dataset{
				Name:        ds.DatasetID,
				ProjectID:   projectID,
				Description: meta.Description,
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "name"}, {Name: "project_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"description": meta.Description,
					"updated_at":  time.Now(),
					"to_delete":   false,
				}),
			}).Create(&dataset).Error; err != nil {
				tx.Rollback()
				ctx.Logger.Error("Failed to create or update dataset", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or update dataset"})
				return
			}

			datasetDoc := utils.DatasetToDocument(&dataset)
			documentsToIndex = append(documentsToIndex, datasetDoc)

			tblIt := ds.Tables(context.Background())
			for {
				tbl, err := tblIt.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					tx.Rollback()
					ctx.Logger.Error("Failed to fetch tables", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tables"})
					return
				}

				tblMeta, err := tbl.Metadata(context.Background())
				if err != nil {
					tx.Rollback()
					ctx.Logger.Error("Failed to fetch table metadata", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch table metadata"})
					return
				}

				table := entity.Table{
					Name:        tbl.TableID,
					DatasetID:   dataset.ID,
					Description: tblMeta.Description,
					RowCount:    tblMeta.NumRows,
				}

				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "name"}, {Name: "dataset_id"}},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"description": tblMeta.Description,
						"row_count":   tblMeta.NumRows,
						"updated_at":  time.Now(),
						"to_delete":   false,
					}),
				}).Create(&table).Error; err != nil {
					tx.Rollback()
					ctx.Logger.Error("Failed to create or update table", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or update table"})
					return
				}

				tableDoc, err := utils.TableToDocument(tx, &table)
				if err != nil {
					ctx.Logger.Error("Failed to create table document", zap.Error(err), zap.String("table_id", table.ID.String()))
				} else {
					documentsToIndex = append(documentsToIndex, tableDoc)
				}

				for _, fieldSchema := range tblMeta.Schema {
					column := entity.Column{
						Name:        fieldSchema.Name,
						Type:        string(fieldSchema.Type),
						Description: fieldSchema.Description,
						TableID:     table.ID,
					}

					if err := tx.Clauses(clause.OnConflict{
						Columns: []clause.Column{{Name: "name"}, {Name: "table_id"}},
						DoUpdates: clause.Assignments(map[string]interface{}{
							"type":        string(fieldSchema.Type),
							"description": fieldSchema.Description,
							"updated_at":  time.Now(),
							"to_delete":   false,
						}),
					}).Create(&column).Error; err != nil {
						tx.Rollback()
						ctx.Logger.Error("Failed to create or update column", zap.Error(err))
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or update column"})
						return
					}

					columnDoc, err := utils.ColumnToDocument(tx, &column)
					if err != nil {
						ctx.Logger.Error("Failed to create column document", zap.Error(err), zap.String("column_id", column.ID.String()))
					} else {
						documentsToIndex = append(documentsToIndex, columnDoc)
					}
				}
			}
		}

		// Handle deletions
		var datasetsToDelete []entity.Dataset
		if err := tx.Where("project_id = ? AND to_delete = ?", projectID, true).Find(&datasetsToDelete).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to fetch datasets to delete", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch datasets to delete"})
			return
		}

		var tablesToDelete []entity.Table
		if err := tx.Where("dataset_id IN (SELECT id FROM datasets WHERE project_id = ?) AND to_delete = ?", projectID, true).Find(&tablesToDelete).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to fetch tables to delete", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tables to delete"})
			return
		}

		var columnsToDelete []entity.Column
		if err := tx.Where("table_id IN (SELECT id FROM tables WHERE dataset_id IN (SELECT id FROM datasets WHERE project_id = ?)) AND to_delete = ?", projectID, true).Find(&columnsToDelete).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to fetch columns to delete", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch columns to delete"})
			return
		}

		if err := tx.Where("project_id = ? AND to_delete = ?", projectID, true).Delete(&entity.Dataset{}).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to delete datasets", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete datasets"})
			return
		}

		if err := tx.Where("dataset_id IN (SELECT id FROM datasets WHERE project_id = ?) AND to_delete = ?", projectID, true).Delete(&entity.Table{}).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to delete tables", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tables"})
			return
		}

		if err := tx.Where("table_id IN (SELECT id FROM tables WHERE dataset_id IN (SELECT id FROM datasets WHERE project_id = ?)) AND to_delete = ?", projectID, true).Delete(&entity.Column{}).Error; err != nil {
			tx.Rollback()
			ctx.Logger.Error("Failed to delete columns", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete columns"})
			return
		}

		if err := tx.Commit().Error; err != nil {
			ctx.Logger.Error("Failed to commit transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		// Batch index documents
		if len(documentsToIndex) > 0 {
			_, err := ctx.MeilisearchClient.Index("resources").AddDocuments(documentsToIndex, "id")
			if err != nil {
				ctx.Logger.Error("Failed to batch index documents", zap.Error(err))
				// Continue execution, as the database transaction was successful
			}
		}

		// Remove deleted documents from index
		var idsToDelete []string
		for _, dataset := range datasetsToDelete {
			idsToDelete = append(idsToDelete, dataset.ID.String())
		}
		for _, table := range tablesToDelete {
			idsToDelete = append(idsToDelete, table.ID.String())
		}
		for _, column := range columnsToDelete {
			idsToDelete = append(idsToDelete, column.ID.String())
		}

		if len(idsToDelete) > 0 {
			_, err := ctx.MeilisearchClient.Index("resources").DeleteDocuments(idsToDelete)
			if err != nil {
				ctx.Logger.Error("Failed to delete documents from index", zap.Error(err))
				// Continue execution, as the database transaction was successful
			}
		}

		syncID := uuid.New()

		newState, err := utils.FetchCurrentState(ctx, projectID)
		if err != nil {
			ctx.Logger.Error("Failed to fetch new state for changelog", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch new state for changelog"})
			return
		}

		sync := entity.Sync{
			ID:        syncID,
			ProjectID: &projectID,
		}
		if err := ctx.DB.Create(&sync).Error; err != nil {
			ctx.Logger.Error("Failed to create sync", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sync"})
			return
		}

		if err := utils.RecordChanges(ctx, syncID, oldState, newState); err != nil {
			ctx.Logger.Error("Failed to record changes for changelog", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record changes for changelog"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Schema fetched and stored successfully"})
	}
}

func GetSyncsByProjectID(ctx *appcontext.Context) gin.HandlerFunc {
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

		var syncs []entity.Sync
		if err := ctx.DB.Where("project_id = ?", projectID).Order("created_at DESC").Limit(5).Find(&syncs).Error; err != nil {
			ctx.Logger.Error("Failed to get syncs from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get syncs from database"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"syncs": syncs})
	}
}

func GetSyncsWithChangelogByProjectID(ctx *appcontext.Context) gin.HandlerFunc {
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

		var syncs []entity.Sync
		if err := ctx.DB.Preload("Changelogs").Where("project_id = ?", projectID).Order("created_at DESC").Find(&syncs).Error; err != nil {
			ctx.Logger.Error("Failed to get syncs with changelogs from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get syncs with changelogs from database"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"syncs": syncs})
	}
}

func GetChangelogsBySyncID(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("projectID")
		syncID := c.Param("syncID")

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

		var changelogs []entity.Changelog
		if err := ctx.DB.Where("sync_id = ?", syncID).Find(&changelogs).Error; err != nil {
			ctx.Logger.Error("Failed to get changelogs by sync ID", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get changelogs by sync ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"changelogs": changelogs})
	}
}
