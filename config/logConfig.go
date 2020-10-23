package config

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

func init() {
	filePath := "/mnt/mmc/lcrtu/log/lcrtu-update.log.%Y%m%d%H%M"
	fileWriter, err := rotatelogs.New(
		filePath,
		rotatelogs.WithLinkName(filePath), // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(time.Hour*24*time.Duration(30)),      // 文件最大保存时间
		rotatelogs.WithRotationTime(time.Hour*24*time.Duration(7)), // 日志切割时间间隔
	)
	if err != nil {
		fmt.Println("初始化日志失败")
		os.Exit(1)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, fileWriter))
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   true,
		FullTimestamp:   true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}
