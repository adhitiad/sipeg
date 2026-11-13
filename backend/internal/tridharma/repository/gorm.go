package repository

import (
	"gorm.io/gorm"
)

// GormStore adalah adapter persistensi Tridharma & BKD berbasis GORM.
type GormStore struct {
	DB *gorm.DB
}
