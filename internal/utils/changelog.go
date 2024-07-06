package utils

import (
	"encoding/json"
	"reflect"

	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func compareAndLogChanges(ctx *appcontext.Context, syncID uuid.UUID, entityType string, entityID uuid.UUID, entityName string, oldEntity, newEntity interface{}, parentID *uuid.UUID, parentName string, grandParentID *uuid.UUID, grandParentName string) {
	oldValue := reflect.ValueOf(oldEntity)
	newValue := reflect.ValueOf(newEntity)

	for i := 0; i < oldValue.NumField(); i++ {
		fieldName := oldValue.Type().Field(i).Name
		if contains([]string{"Model", "Columns", "Tables"}, fieldName) {
			continue
		}
		oldFieldValue := oldValue.Field(i).Interface()
		newFieldValue := newValue.Field(i).Interface()

		if !reflect.DeepEqual(oldFieldValue, newFieldValue) {
			changelog := entity.Changelog{
				ChangeType:      "update",
				EntityType:      entityType,
				EntityID:        entityID,
				EntityName:      entityName,
				FieldName:       fieldName,
				OldValue:        toJSON(oldFieldValue),
				NewValue:        toJSON(newFieldValue),
				ParentID:        parentID,
				ParentName:      parentName,
				GrandParentID:   grandParentID,
				GrandParentName: grandParentName,
				SyncID:          &syncID,
			}
			ctx.DB.Create(&changelog)
		}
	}
}

func FetchCurrentState(ctx *appcontext.Context, projectID uuid.UUID) ([]entity.Dataset, error) {
	var datasets []entity.Dataset
	if err := ctx.DB.Preload("Tables.Columns").Where("project_id = ?", projectID).Find(&datasets).Error; err != nil {
		return nil, err
	}
	return datasets, nil
}

func RecordChanges(ctx *appcontext.Context, syncID uuid.UUID, oldDatasets, newDatasets []entity.Dataset) error {

	oldDatasetsMap := make(map[uuid.UUID]entity.Dataset)
	newDatasetsMap := make(map[uuid.UUID]entity.Dataset)
	for _, ds := range oldDatasets {
		oldDatasetsMap[ds.ID] = ds
	}
	for _, ds := range newDatasets {
		newDatasetsMap[ds.ID] = ds
	}

	for id, newDs := range newDatasetsMap {
		oldDs, exists := oldDatasetsMap[id]
		if !exists {
			changelog := entity.Changelog{
				ChangeType: "insert",
				EntityType: "dataset",
				EntityID:   newDs.ID,
				EntityName: newDs.Name,
				FieldName:  "",
				OldValue:   "",
				NewValue:   "",
				ParentName: "",
				SyncID:     &syncID,
			}
			ctx.DB.Create(&changelog)
		} else {
			compareAndLogChanges(ctx, syncID, "dataset", newDs.ID, newDs.Name, oldDs, newDs, nil, "", nil, "")
			compareTablesAndLogChanges(ctx, syncID, oldDs.Tables, newDs.Tables, newDs.ID, newDs.Name)
		}
		delete(oldDatasetsMap, id)
	}

	for _, oldDs := range oldDatasetsMap {
		changelog := entity.Changelog{
			ChangeType: "delete",
			EntityType: "dataset",
			EntityID:   oldDs.ID,
			EntityName: oldDs.Name,
			FieldName:  "",
			OldValue:   "",
			NewValue:   "",
			ParentName: "",
			SyncID:     &syncID,
		}
		ctx.DB.Create(&changelog)
	}

	return nil
}

func compareTablesAndLogChanges(ctx *appcontext.Context, syncID uuid.UUID, oldTables, newTables []entity.Table, datasetID uuid.UUID, datasetName string) {
	oldTablesMap := make(map[uuid.UUID]entity.Table)
	newTablesMap := make(map[uuid.UUID]entity.Table)
	for _, tbl := range oldTables {
		oldTablesMap[tbl.ID] = tbl
	}
	for _, tbl := range newTables {
		newTablesMap[tbl.ID] = tbl
	}

	for id, newTbl := range newTablesMap {
		oldTbl, exists := oldTablesMap[id]
		if !exists {
			changelog := entity.Changelog{
				ChangeType: "insert",
				EntityType: "table",
				EntityID:   newTbl.ID,
				EntityName: newTbl.Name,
				FieldName:  "",
				OldValue:   "",
				NewValue:   "",
				ParentID:   &datasetID,
				ParentName: datasetName,
				SyncID:     &syncID,
			}
			ctx.DB.Create(&changelog)
		} else {
			compareAndLogChanges(ctx, syncID, "table", newTbl.ID, newTbl.Name, oldTbl, newTbl, &datasetID, datasetName, nil, "")
			compareColumnsAndLogChanges(ctx, syncID, oldTbl.Columns, newTbl.Columns, newTbl.ID, newTbl.Name, datasetID, datasetName)
		}
		delete(oldTablesMap, id)
	}

	for _, oldTbl := range oldTablesMap {
		changelog := entity.Changelog{
			ChangeType: "delete",
			EntityType: "table",
			EntityID:   oldTbl.ID,
			EntityName: oldTbl.Name,
			FieldName:  "",
			OldValue:   "",
			NewValue:   "",
			ParentID:   &datasetID,
			ParentName: datasetName,
			SyncID:     &syncID,
		}
		ctx.DB.Create(&changelog)
	}
}

func compareColumnsAndLogChanges(ctx *appcontext.Context, syncID uuid.UUID, oldColumns, newColumns []entity.Column, tableID uuid.UUID, tableName string, datasetID uuid.UUID, datasetName string) {
	oldColumnsMap := make(map[uuid.UUID]entity.Column)
	newColumnsMap := make(map[uuid.UUID]entity.Column)
	for _, col := range oldColumns {
		oldColumnsMap[col.ID] = col
	}
	for _, col := range newColumns {
		newColumnsMap[col.ID] = col
	}

	for id, newCol := range newColumnsMap {
		oldCol, exists := oldColumnsMap[id]
		if !exists {
			changelog := entity.Changelog{
				ChangeType:      "insert",
				EntityType:      "column",
				EntityID:        newCol.ID,
				EntityName:      newCol.Name,
				FieldName:       "",
				OldValue:        "",
				NewValue:        "",
				ParentID:        &tableID,
				ParentName:      tableName,
				GrandParentID:   &datasetID,
				GrandParentName: datasetName,
				SyncID:          &syncID,
			}
			ctx.DB.Create(&changelog)
		} else {
			compareAndLogChanges(ctx, syncID, "column", newCol.ID, newCol.Name, oldCol, newCol, &tableID, tableName, &datasetID, datasetName)
		}
		delete(oldColumnsMap, id)
	}

	for _, oldCol := range oldColumnsMap {
		changelog := entity.Changelog{
			ChangeType:      "delete",
			EntityType:      "column",
			EntityID:        oldCol.ID,
			EntityName:      oldCol.Name,
			FieldName:       "",
			OldValue:        "",
			NewValue:        "",
			ParentID:        &tableID,
			ParentName:      tableName,
			GrandParentID:   &datasetID,
			GrandParentName: datasetName,
			SyncID:          &syncID,
		}
		ctx.DB.Create(&changelog)
	}
}

func toJSON(v interface{}) string {
	jsonBytes, _ := json.Marshal(v)
	return string(jsonBytes)
}
