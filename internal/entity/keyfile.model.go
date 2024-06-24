package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KeyFile struct {
	gorm.Model
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	UserID uuid.UUID `gorm:"type:uuid;not null"`
	URL    string    `gorm:"type:text;not null"`
}
