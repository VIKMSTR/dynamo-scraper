package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	loggingToFile := os.Getenv("SCRAPING_LOG_TO_FILE")
	if loggingToFile == "" {
		loggingToFile = "false"
	}
	loglevel := os.Getenv("SCRAPING_LOG_LEVEL")
	if loglevel == "" {
		loglevel = "info"
	}
	if strings.ToLower(loggingToFile) == "true" {
		file, err := os.OpenFile("scraper.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.Out = file
		} else {
			log.Info("Failed to log to file, using default stderr")
		}
	}
	switch strings.ToLower(loglevel) {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	case "fatal":
		log.SetLevel(logrus.FatalLevel)
	case "panic":
		log.SetLevel(logrus.PanicLevel)
	default:
		log.SetLevel(logrus.FatalLevel)
	}
}
