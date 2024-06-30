package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Project struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Name      string    `gorm:"type:varchar(100);unique_index"`
	Datasets  []Dataset `gorm:"foreignKey:ProjectID"`
	KeyFile   *KeyFile  `gorm:"foreignKey:ProjectID"`
	Syncs     []Sync    `gorm:"foreignKey:ProjectID"`
	CompanyID uuid.UUID `gorm:"type:uuid"`
	Company   Company   `gorm:"foreignKey:CompanyID"`
}
