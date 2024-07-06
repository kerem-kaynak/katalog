package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Column struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_column_name_table" json:"name"`
	Type        string    `gorm:"type:varchar(255);not null" json:"type"`
	Description string    `gorm:"type:text" json:"description"`
	TableID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_column_name_table" json:"table_id"`
	ToDelete    bool      `gorm:"type:boolean" json:"to_delete"`
}
