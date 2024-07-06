package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Changelog struct {
	gorm.Model
	ID              uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	ChangeType      string     `gorm:"type:varchar(100)" json:"change_type"`
	EntityType      string     `gorm:"type:varchar(100)" json:"entity_type"`
	EntityID        uuid.UUID  `gorm:"type:uuid" json:"entity_id"`
	EntityName      string     `gorm:"type:varchar(255)" json:"entity_name"`
	FieldName       string     `gorm:"type:varchar(255)" json:"field_name"`
	OldValue        string     `gorm:"type:text" json:"old_value"`
	NewValue        string     `gorm:"type:text" json:"new_value"`
	ParentID        *uuid.UUID `gorm:"type:uuid" json:"parent_id"`
	ParentName      string     `gorm:"type:varchar(255)" json:"parent_name"`
	GrandParentID   *uuid.UUID `gorm:"type:uuid" json:"grandparent_id"`
	GrandParentName string     `gorm:"type:varchar(255)" json:"grandparent_name"`
	SyncID          *uuid.UUID `gorm:"type:uuid" json:"sync_id"`
}
