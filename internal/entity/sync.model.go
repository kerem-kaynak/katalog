package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Sync struct {
	gorm.Model
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	ProjectID   *uuid.UUID `gorm:"type:uuid;not null" json:"project_id"`
	ChangelogID *uuid.UUID `gorm:"type:uuid" json:"changelog_id"`
}
