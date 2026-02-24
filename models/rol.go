package models

type Rol struct {
	ID     uint   `gorm:"primaryKey"`
	Nombre string `gorm:"size:50;unique;not null"`

	Usuarios []Usuario `gorm:"foreignKey:RolID"`
}
