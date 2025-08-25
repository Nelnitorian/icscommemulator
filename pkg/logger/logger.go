package logger

import (
	"log"
	"os"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARNING
	ERROR
)

var (
	currentLevel = INFO
	debugLogger  = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	infoLogger   = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	warnLogger   = log.New(os.Stdout, "[WARNING] ", log.LstdFlags)
	errorLogger  = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile)
)

// SetLevel configura el nivel mínimo de logging
func SetLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARNING", "WARN":
		currentLevel = WARNING
	case "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
	Info("Log level set to: %s", strings.ToUpper(level))
}

func Debug(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		debugLogger.Printf(format, v...)
	}
}

func Info(format string, v ...interface{}) {
	if currentLevel <= INFO {
		infoLogger.Printf(format, v...)
	}
}

func Warning(format string, v ...interface{}) {
	if currentLevel <= WARNING {
		warnLogger.Printf(format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		errorLogger.Printf(format, v...)
	}
}

func DebugStruct(name string, obj interface{}) {
	if currentLevel <= DEBUG {
		debugLogger.Printf("%s: %+v", name, obj)
	}
}
