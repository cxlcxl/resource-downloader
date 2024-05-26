package spider

import "gorm.io/gorm"

type Option func(*Spider)

func SetAsync(limitGos int) Option {
	return func(s *Spider) {
		s.LimitGos = limitGos
		s.Async = true
	}
}

func SetBodyMaxSize(maxSize int) Option {
	return func(s *Spider) {
		s.maxSize = maxSize
	}
}

func SetUserAgent(us string) Option {
	return func(s *Spider) {
		s.userAgent = us
	}
}

func SetOnce(onceUrl string) Option {
	return func(s *Spider) {
		s.isOnce = true
		s.onceUrl = onceUrl
	}
}

func UseDb(db *gorm.DB) Option {
	return func(s *Spider) {
		s.db = db
	}
}
