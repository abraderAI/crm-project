package models

// APIUsageStat tracks per-endpoint request counts rolled up hourly.
type APIUsageStat struct {
	Endpoint string `gorm:"type:text;primaryKey" json:"endpoint"`
	Method   string `gorm:"type:text;primaryKey" json:"method"`
	Hour     string `gorm:"type:text;primaryKey" json:"hour"` // YYYY-MM-DD-HH
	Count    int64  `gorm:"default:0" json:"count"`
}
