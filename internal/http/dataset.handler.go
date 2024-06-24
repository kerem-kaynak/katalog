package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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

func FetchDatasets(ctx *appcontext.Context) gin.HandlerFunc {
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

		var keyFile entity.KeyFile
		if err := ctx.DB.Where("user_id = ?", userID).First(&keyFile).Error; err != nil {
			ctx.Logger.Error("Failed to get key file for user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get key file for user"})
			return
		}

		// Fetch the key file from GCS
		rc, err := ctx.GCSClient.Bucket(ctx.GCSBucketName).Object(userID.String() + "/sa_key").NewReader(context.Background())
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

		// Use the key file to authenticate and fetch datasets from BigQuery
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

		it := client.Datasets(context.Background())
		var datasets []entity.Dataset
		for {
			ds, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				ctx.Logger.Error("Failed to fetch datasets", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch datasets"})
				return
			}

			meta, err := ds.Metadata(context.Background())
			if err != nil {
				ctx.Logger.Error("Failed to fetch dataset metadata", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dataset metadata"})
				return
			}

			dataset := entity.Dataset{
				Name:        ds.DatasetID,
				ProjectID:   ds.ProjectID,
				Description: meta.Description,
				CompanyID:   user.CompanyID,
			}

			if err := ctx.DB.Create(&dataset).Error; err != nil {
				ctx.Logger.Error("Failed to store dataset in database", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store dataset in database"})
				return
			}
			datasets = append(datasets, dataset)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Datasets fetched and stored successfully", "datasets": datasets})
	}
}
