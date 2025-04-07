package models

type Streamers struct {
	ID            int    `gorm:"primaryKey;column:id"`
	Platform      string `gorm:"column:platform;type:varchar(50);not null"`
	Username      string `gorm:"column:username;type:varchar(100);not null"`
	Quality       string `gorm:"column:quality;type:varchar(20);not null"`
	SplitSegments bool   `gorm:"column:split_segments;not null"`
	TimeSegment   int    `gorm:"column:time_segment;not null"`
}
