package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Company struct {
	gorm.Model
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	Name     string    `gorm:"type:varchar(100);unique_index"`
	Users    []User    `gorm:"foreignKey:CompanyID"`
	Datasets []Dataset `gorm:"foreignKey:CompanyID"`
}
