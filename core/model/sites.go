package model

import (
	"gorm.io/gorm"
	"time"
)

type Site struct {
	SiteEnCode   string    `gorm:"column:site_en_code"`
	SiteUrl      string    `gorm:"column:site_url"`
	State        uint8     `gorm:"column:state"`
	ScheduleSpec string    `gorm:"column:schedule_spec"`
	LastScrapyAt time.Time `gorm:"column:last_scrapy_at"`
	Version      uint64    `gorm:"column:version"`
	Timestamp
}

func NewSite() *Site {
	return &Site{}
}

func (m *Site) TableName() string {
	return "sites"
}

func (m *Site) FindSiteJobs(db *gorm.DB) (sites []*Site, err error) {
	err = db.Table(m.TableName()).
		Where("state = 1").
		Select("site_en_code", "site_url", "schedule_spec", "version").
		Find(&sites).Error
	return
}
