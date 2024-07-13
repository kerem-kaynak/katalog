package utils

import (
	"fmt"

	"github.com/kerem-kaynak/katalog/internal/entity"
	"gorm.io/gorm"
)

func DatasetToDocument(dataset *entity.Dataset) map[string]interface{} {
	return map[string]interface{}{
		"id":          dataset.ID.String(),
		"type":        "dataset",
		"name":        dataset.Name,
		"description": dataset.Description,
		"project_id":  dataset.ProjectID.String(),
	}
}

func TableToDocument(db *gorm.DB, table *entity.Table) (map[string]interface{}, error) {
	var dataset entity.Dataset
	if err := db.First(&dataset, table.DatasetID).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch dataset for table: %w", err)
	}

	return map[string]interface{}{
		"id":           table.ID.String(),
		"type":         "table",
		"name":         table.Name,
		"description":  table.Description,
		"row_count":    table.RowCount,
		"project_id":   dataset.ProjectID.String(),
		"parent_id":    table.DatasetID.String(),
		"dataset_id":   table.DatasetID.String(),
		"dataset_name": dataset.Name,
	}, nil
}

func ColumnToDocument(db *gorm.DB, column *entity.Column) (map[string]interface{}, error) {
	var table entity.Table
	if err := db.First(&table, column.TableID).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch table for column: %w", err)
	}

	var dataset entity.Dataset
	if err := db.First(&dataset, table.DatasetID).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch dataset for column: %w", err)
	}

	return map[string]interface{}{
		"id":           column.ID.String(),
		"type":         "column",
		"name":         column.Name,
		"description":  column.Description,
		"column_type":  column.Type,
		"project_id":   dataset.ProjectID.String(),
		"parent_id":    column.TableID.String(),
		"table_id":     column.TableID.String(),
		"dataset_id":   table.DatasetID.String(),
		"table_name":   table.Name,
		"dataset_name": dataset.Name,
	}, nil
}

// func IndexDocument(ctx *appcontext.Context, document map[string]interface{}) error {
// 	_, err := ctx.MeilisearchClient.Index("resources").AddDocuments([]map[string]interface{}{document})
// 	if err != nil {
// 		return fmt.Errorf("failed to index document: %w", err)
// 	}
// 	return nil
// }

// func RemoveDocument(ctx *appcontext.Context, id string) error {
// 	_, err := ctx.MeilisearchClient.Index("resources").DeleteDocument(id)
// 	if err != nil {
// 		return fmt.Errorf("failed to remove document: %w", err)
// 	}
// 	return nil
// }
