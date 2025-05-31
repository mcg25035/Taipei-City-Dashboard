package models

import (
	"time"
)

// HomeDownEvent represents a user-reported home down event.
type HomeDownEvent struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Latitude    float64   `gorm:"type:decimal(10,8);not null" json:"latitude"`
	Longitude   float64   `gorm:"type:decimal(11,8);not null" json:"longitude"`
	Type        string    `gorm:"type:varchar(50);not null" json:"type"`
	Name        *string   `gorm:"type:varchar(255)" json:"name"` // Pointer to allow NULL
	ReportedAt  time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"reported_at"`
}

// TableName specifies the table name for HomeDownEvent.
func (HomeDownEvent) TableName() string {
	return "public.home_down_events"
}
