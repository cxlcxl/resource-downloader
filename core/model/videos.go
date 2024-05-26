package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Video struct {
	VideoId     string `json:"video_id"`
	VideoName   string `json:"video_name"`
	VideoDesc   string `json:"video_desc"`
	CoverImg    string `json:"cover_img"`
	Actors      string `json:"actors"`
	Directors   string `json:"directors"`
	OnlineDate  string `json:"online_date"`
	Episodes    string `json:"episodes"`
	State       uint8  `json:"state"`
	FromSiteUrl string `json:"from_site_url"`

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
