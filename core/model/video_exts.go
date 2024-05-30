package model

import (
	"fmt"
	"gorm.io/gorm"
)

type VideoExt struct {
	VideoId   string `gorm:"column:video_id"`
	ExtKey    string `gorm:"column:ext_key"`
	ExtVal    string `gorm:"column:ext_val"`
	State     uint8  `gorm:"column:state"`
	ExtDetail string `gorm:"column:ext_detail"`
}

func NewVideoExt() *VideoExt {
	return &VideoExt{}
}

func (m *VideoExt) TableName() string {
	return "video_exts"
}

func (m *VideoExt) CreateVideoExts(db *gorm.DB, videoId string, exts []*VideoExt) (err error) {
	return db.Transaction(func(tx *gorm.DB) error {
		if err = tx.Exec(fmt.Sprintf("delete from %s where video_id = ?", m.TableName()), videoId).Error; err != nil {
			return err
		}
		if err = tx.Table(m.TableName()).CreateInBatches(exts, 100).Error; err != nil {
			return err
		}
		return nil
	})
}
