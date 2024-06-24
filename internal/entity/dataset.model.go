package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Dataset struct {
	gorm.Model
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Name        string     `gorm:"type:varchar(100);uniqueIndex:idx_dataset_name_company" json:"name"`
	ProjectID   string     `gorm:"type:varchar(100)" json:"project_id"`
	Description string     `gorm:"type:text" json:"description"`
	CompanyID   *uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_dataset_name_company" json:"company_id"`
	Tables      []Table    `gorm:"foreignKey:DatasetID" json:"tables"`
}
