package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Video struct {
	VideoId     string `gorm:"column:video_id"`
	VideoName   string `gorm:"column:video_name"`
	VideoDesc   string `gorm:"column:video_desc"`
	CoverImg    string `gorm:"column:cover_img"`
	Actors      string `gorm:"column:actors"`
	Directors   string `gorm:"column:directors"`
	OnlineDate  string `gorm:"column:online_date"`
	Episodes    string `gorm:"column:episodes"`
	State       uint8  `gorm:"column:state"`
	FromSiteUrl string `gorm:"column:from_site_url"`

	Timestamp
}

func NewVideo() *Video {
	return &Video{}
}

func (m *Video) TableName() string {
	return "videos"
}

func (m *Video) CreateVideo(db *gorm.DB, video *Video, exts []*VideoExt) error {
	err := db.Table(m.TableName()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "video_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"video_desc", "cover_img", "actors", "directors", "from_site_url", "online_date"}),
	}).Create(video).Error
	if err == nil {
		err = NewVideoExt().CreateVideoExts(db, video.VideoId, exts)
	}
	return err
}
