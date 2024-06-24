package http

import (
	"context"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"

	"net/http"
)

func UploadFile(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID from claims"})
			return
		}

		bucketName := ctx.GCSBucketName

		file, err := c.FormFile("file")
		if err != nil {
			ctx.Logger.Error("Failed to get file from request", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request"})
			return
		}

		if !isJSONFile(file) {
			ctx.Logger.Error("Invalid file type")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type, only JSON files are allowed"})
			return
		}

		src, err := file.Open()
		if err != nil {
			ctx.Logger.Error("Failed to open file", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer src.Close()

		objectPath := userID.String() + "/" + "sa_key"

		w := ctx.GCSClient.Bucket(bucketName).Object(objectPath).NewWriter(context.Background())

		if _, err := io.Copy(w, src); err != nil {
			ctx.Logger.Error("Failed to upload file to GCS: " + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to GCS"})
			return
		}

		if err := w.Close(); err != nil {
			ctx.Logger.Error("Failed to close GCS writer: " + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close GCS writer"})
			return
		}

		fileURL := "https://storage.googleapis.com/" + bucketName + "/" + objectPath

		keyFile := entity.KeyFile{
			UserID: userID,
			URL:    fileURL,
		}

		if err := ctx.DB.Where("user_id = ?", userID).Delete(&entity.KeyFile{}).Error; err != nil {
			ctx.Logger.Error("Failed to delete existing key file", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing key file"})
			return
		}

		if err := ctx.DB.Create(&keyFile).Error; err != nil {
			ctx.Logger.Error("Failed to store key file URL in database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store key file URL in database"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
	}
}

func isJSONFile(file *multipart.FileHeader) bool {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".json" {
		return false
	}

	mimeType := file.Header.Get("Content-Type")
	return mimeType == "application/json" || mimeType == "application/octet-stream"
}
