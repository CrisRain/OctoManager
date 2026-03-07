package model

import "time"

type BaseModel struct {
    ID        uint64    `gorm:"primaryKey;autoIncrement;type:bigint" json:"id"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type BaseID struct {
    ID uint64 `gorm:"primaryKey;autoIncrement;type:bigint" json:"id"`
}

type BaseTimestamps struct {
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
