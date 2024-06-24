package appcontext

import (
	"cloud.google.com/go/storage"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Context struct {
	DB     *gorm.DB
	Logger *zap.Logger

	GCSClient     *storage.Client
	GCPProjectID  string
	GCSBucketName string

	OAuth2Config *oauth2.Config
}
