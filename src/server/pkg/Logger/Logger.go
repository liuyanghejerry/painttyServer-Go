package logger

import (
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
)

func SetupLogs(logName string) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(&lumberjack.Logger{
		Filename:   fmt.Sprintf("./logs/%s.log", logName),
		MaxSize:    2, // megabytes
		MaxBackups: 3,
		MaxAge:     60,    //days
		Compress:   false, // disabled by default
	})
}
