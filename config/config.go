package config

type Config struct {
	Logs  *Logs  `yaml:"logs"`
	Video *Video `yaml:"video"`
}

type Logs struct {
	Dir        string `yaml:"dir"`
	LogName    string `yaml:"log_name"`
	MaxBackups int    `yaml:"max_backups"`
}

type Video struct {
	SavePath string `yaml:"save_path"`
}
