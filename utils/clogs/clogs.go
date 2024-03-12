package clogs

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"io"
	"path"
	"time"
	"videocapture/vars"
)

type LogInterface interface {
	InfoLog(map[string]interface{}, string)
	WarnLog(map[string]interface{}, string)
	ErrLog(map[string]interface{}, string)
	DebugLog(map[string]interface{}, string)
}

type CLog struct {
	*logrus.Logger
}

// NewCLog 初始化系统日志
func NewCLog() *CLog {
	clog := &CLog{logrus.New()}
	//clog.SetReportCaller(true) // 添加调用的函数和文件

	conf := vars.Config.Logs
	logName := path.Join(
		vars.BasePath,
		conf.Dir,
		conf.LogName,
	)
	defaultWriter := getWriter(logName, "info", conf.MaxBackups)
	errWriter := getWriter(logName, "error", conf.MaxBackups)
	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: defaultWriter,
		logrus.InfoLevel:  defaultWriter,
		logrus.WarnLevel:  defaultWriter,
		logrus.ErrorLevel: errWriter,
		logrus.FatalLevel: errWriter,
		logrus.PanicLevel: errWriter,
	}, &logrus.JSONFormatter{TimestampFormat: vars.DateTimeFormat})

	clog.AddHook(lfsHook)

	return clog
}

func getWriter(logName, level string, maxBackups int) io.Writer {
	writer, _ := rotatelogs.New(
		logName+"-"+level+".%Y%m%d",
		// WithLinkName为最新的日志建立软连接，以方便随着找到当前日志文件
		rotatelogs.WithLinkName(logName),

		// WithRotationTime设置日志分割的时间
		rotatelogs.WithRotationTime(time.Hour*24),

		// WithMaxAge和WithRotationCount二者只能设置一个，
		// WithMaxAge设置文件清理前的最长保存时间，
		// WithRotationCount设置文件清理前最多保存的个数。
		//rotatelogs.WithMaxAge(time.Duration(vars.YmlConfig.GetInt("Logs.MaxAge"))*time.Second*3600*24),
		rotatelogs.WithRotationCount(uint(maxBackups)),
	)
	return writer
}

func (l *CLog) InfoLog(logs map[string]interface{}, prefix string) {
	l.WithField(prefix, logs).Log(logrus.InfoLevel)
}

func (l *CLog) WarnLog(logs map[string]interface{}, prefix string) {
	l.WithField(prefix, logs).Log(logrus.WarnLevel)
}

func (l *CLog) ErrLog(logs map[string]interface{}, prefix string) {
	l.WithField(prefix, logs).Log(logrus.ErrorLevel)
}

func (l *CLog) DebugLog(logs map[string]interface{}, prefix string) {
	l.WithField(prefix, logs).Log(logrus.DebugLevel)
}
