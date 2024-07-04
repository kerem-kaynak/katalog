package utils

import (
	"github.com/google/uuid"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
)

func UserHasProjectAccess(ctx *appcontext.Context, userID uuid.UUID, projectID uuid.UUID) bool {
	var user entity.User
	var project entity.Project

	if err := ctx.DB.Preload("Company").First(&user, userID).Error; err != nil {
		return false
	}

	if err := ctx.DB.Where("id = ? AND company_id = ?", projectID, user.CompanyID).First(&project).Error; err != nil {
		return false
	}

	return true
}

func UserHasDatasetAccess(ctx *appcontext.Context, userID uuid.UUID, datasetID uuid.UUID) bool {
	var user entity.User
	var dataset entity.Dataset
	var project entity.Project

	if err := ctx.DB.Preload("Company").First(&user, userID).Error; err != nil {
		return false
	}

	if err := ctx.DB.First(&dataset, datasetID).Error; err != nil {
		return false
	}

	if err := ctx.DB.Where("id = ? AND company_id = ?", dataset.ProjectID, user.CompanyID).First(&project).Error; err != nil {
		return false
	}

	return true
}

func UserHasTableAccess(ctx *appcontext.Context, userID uuid.UUID, tableID uuid.UUID) bool {
	var user entity.User
	var table entity.Table
	var dataset entity.Dataset
	var project entity.Project

	if err := ctx.DB.Preload("Company").First(&user, userID).Error; err != nil {
		return false
	}

	if err := ctx.DB.First(&table, tableID).Error; err != nil {
		return false
	}

	if err := ctx.DB.First(&dataset, table.DatasetID).Error; err != nil {
		return false
	}

	if err := ctx.DB.Where("id = ? AND company_id = ?", dataset.ProjectID, user.CompanyID).First(&project).Error; err != nil {
		return false
	}

	return true
}
