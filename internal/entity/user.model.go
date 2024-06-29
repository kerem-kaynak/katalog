package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Email          string     `json:"email" gorm:"type:varchar(100);unique_index"`
	Name           string     `json:"name" gorm:"type:varchar(100)"`
	KeyFile        *KeyFile   `json:"key_file" gorm:"foreignkey:UserID;constraint:OnDelete:CASCADE"`
	CompanyID      *uuid.UUID `json:"company_id" gorm:"type:uuid"`
	ProfilePicture string     `json:"profile_picture" gorm:"type:varchar(255)"`
	Role           string     `json:"role" gorm:"type:varchar(100)"`
}
