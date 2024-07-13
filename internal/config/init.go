package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"context"

	"cloud.google.com/go/storage"
	"github.com/joho/godotenv"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitContext() (*appcontext.Context, error) {
	if err := godotenv.Load(); err != nil {
		zap.L().Warn("No .env file found, using environment variables")
	}

	logger, err := InitLogger()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()

	db, err := InitDB()
	if err != nil {
		return nil, err
	}

	gcsClient, err := InitGCSClient()
	if err != nil {
		return nil, err
	}

	meilisearchClient, err := InitMeilisearch()
	if err != nil {
		return nil, err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	ctx := &appcontext.Context{
		DB:     db,
		Logger: logger,

		GCSClient:     gcsClient,
		GCPProjectID:  os.Getenv("GCP_PROJECT_ID"),
		GCSBucketName: os.Getenv("GCS_BUCKET_NAME"),

		OAuth2Config:      oauth2Config,
		MeilisearchClient: meilisearchClient,
	}

	return ctx, nil
}

func InitDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	err = db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error
	if err != nil {
		return nil, fmt.Errorf("failed to enable uuid-ossp extension: %w", err)
	}

	err = db.AutoMigrate(&entity.Company{}, &entity.User{}, &entity.KeyFile{}, &entity.Dataset{}, &entity.Table{}, &entity.Column{}, &entity.Sync{}, &entity.Project{}, &entity.Changelog{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func InitLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return logger, nil
}

func InitGCSClient() (*storage.Client, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCS client: %w", err)
	}
	return client, nil
}

func InitMeilisearch() (*meilisearch.Client, error) {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host: "http://host.docker.internal:7700",
	})

	_, err := client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        "resources",
		PrimaryKey: "id",
	})
	if err != nil {
		// If the error is because the index already exists, that's fine
		if strings.Contains(err.Error(), "already exists") {
			// Index already exists, continue with updating settings
		} else {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Set filterable attributes
	task, err := client.Index("resources").UpdateFilterableAttributes(&[]string{
		"project_id",
		"type",
		"dataset_id",
		"table_id",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update filterable attributes: %w", err)
	}

	// Wait for the task to complete
	_, err = client.WaitForTask(task.TaskUID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for filterable attributes update: %w", err)
	}

	// Set searchable attributes
	task, err = client.Index("resources").UpdateSearchableAttributes(&[]string{
		"name",
		"description",
		"type",
		"column_type",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update searchable attributes: %w", err)
	}

	// Wait for the task to complete
	_, err = client.WaitForTask(task.TaskUID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for searchable attributes update: %w", err)
	}

	return client, nil
}
