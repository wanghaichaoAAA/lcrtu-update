package config

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   true,
		FullTimestamp:   true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}
