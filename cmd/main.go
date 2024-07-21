package main

import (
	"fmt"
	"log"

	"github.com/kerem-kaynak/katalog/internal/config"
	"github.com/kerem-kaynak/katalog/internal/http"
	"go.uber.org/zap"
)

func main() {
	// Initialize context
	ctx, err := config.InitContext()
	if err != nil {
		log.Fatalf("Failed to initialize context: %v", err)
	}

	defer func() {
		if err := ctx.Logger.Sync(); err != nil {
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Ensure the database connection is closed when the application exits
	sqlDB, err := ctx.DB.DB()
	if err != nil {
		ctx.Logger.Fatal("Failed to get underlying SQL DB from GORM DB", zap.Error(err))
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			ctx.Logger.Fatal("Failed to close database connection", zap.Error(err))
		}
	}()

	// Initialize HTTP service
	service := http.NewHTTPService(ctx)

	// Start the server
	if err := service.Engine().Run(":8080"); err != nil {
		ctx.Logger.Fatal("Failed to start the server", zap.Error(err))
	}
}
