package models

import (
	"time"
)

// InfraDownEvent represents an infrastructure down event announcement.
type InfraDownEvent struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      string    `gorm:"type:varchar(50);not null" json:"type"`
	StartTime time.Time `gorm:"type:timestamp with time zone;not null" json:"start_time"`
	EndTime   time.Time `gorm:"type:timestamp with time zone;not null" json:"end_time"`
	AreaData  string    `gorm:"type:text" json:"area_data"` // Changed to string and type:text
	CreatedAt time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName specifies the table name for InfraDownEvent.
func (InfraDownEvent) TableName() string {
	return "public.infra_down_events"
}
