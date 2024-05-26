package model

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
	"videocapture/vars"
)

type Timestamp struct {
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func NewDB() (db *gorm.DB, err error) {
	db, err = gorm.Open(mysql.Open(vars.Config.Db.Dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		//Logger:                 nil,
	})
	if err != nil {
		return nil, err
	}
	_ = db.Callback().Query().Before("gorm:query").Register("disable_raise_record_not_found", func(gormDB *gorm.DB) {
		gormDB.Statement.RaiseErrorOnNotFound = false
	})
	d, err := db.DB()
	if err != nil {
		return nil, err
	}

	d.SetConnMaxIdleTime(time.Second * 30)  // 最大空闲时间
	d.SetConnMaxLifetime(120 * time.Second) // 最大连接时间
	d.SetMaxIdleConns(10)                   // 最大空闲连接
	d.SetMaxOpenConns(128)                  // 最大连接
	return
}
