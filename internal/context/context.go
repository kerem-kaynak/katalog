package context

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Context struct {
	DB     *gorm.DB
	Logger *zap.Logger
}
