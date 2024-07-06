package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Project struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	Datasets  []Dataset `gorm:"foreignKey:ProjectID" json:"datasets"`
	KeyFile   *KeyFile  `gorm:"foreignKey:ProjectID" json:"key_file"`
	Syncs     []Sync    `gorm:"foreignKey:ProjectID" json:"syncs"`
	CompanyID uuid.UUID `gorm:"type:uuid" json:"company_id"`
	Company   Company   `gorm:"foreignKey:CompanyID" json:"company"`
}
