package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Table struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_table_name_dataset" json:"name"`
	DatasetID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_table_name_dataset" json:"dataset_id"`
	Description string    `gorm:"type:text" json:"description"`
	RowCount    uint64    `gorm:"type:bigint" json:"row_count"`
	Columns     []Column  `gorm:"foreignKey:TableID" json:"columns"`
	ToDelete    bool      `gorm:"type:boolean" json:"to_delete"`
}
