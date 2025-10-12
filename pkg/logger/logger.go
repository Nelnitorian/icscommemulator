package logger

import (
	"context"
	"fmt"
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

// Logger interface permite testing y diferentes implementaciones
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warning(format string, v ...interface{})
	Error(format string, v ...interface{})
	DebugStruct(name string, obj interface{})
	WithContext(ctx context.Context) Logger
	WithFields(fields map[string]interface{}) Logger
}

type logger struct {
	currentLevel Level
	debugLogger  *log.Logger
	infoLogger   *log.Logger
	warnLogger   *log.Logger
	errorLogger  *log.Logger
	fields       map[string]interface{}
}

var defaultLogger = &logger{
	currentLevel: INFO,
	debugLogger:  log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
	infoLogger:   log.New(os.Stdout, "[INFO] ", log.LstdFlags),
	warnLogger:   log.New(os.Stdout, "[WARNING] ", log.LstdFlags),
	errorLogger:  log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	fields:       make(map[string]interface{}),
}

// SetLevel configura el nivel mínimo de logging
func SetLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		defaultLogger.currentLevel = DEBUG
	case "INFO":
		defaultLogger.currentLevel = INFO
	case "WARNING", "WARN":
		defaultLogger.currentLevel = WARNING
	case "ERROR":
		defaultLogger.currentLevel = ERROR
	default:
		defaultLogger.currentLevel = INFO
	}
	Info("Log level set to: %s", strings.ToUpper(level))
}

func (l *logger) formatMessage(format string) string {
	if len(l.fields) == 0 {
		return format
	}
	
	fieldsStr := ""
	for k, v := range l.fields {
		fieldsStr += fmt.Sprintf(" %s=%v", k, v)
	}
	return format + fieldsStr
}

func (l *logger) Debug(format string, v ...interface{}) {
	if l.currentLevel <= DEBUG {
		l.debugLogger.Printf(l.formatMessage(format), v...)
	}
}

func (l *logger) Info(format string, v ...interface{}) {
	if l.currentLevel <= INFO {
		l.infoLogger.Printf(l.formatMessage(format), v...)
	}
}

func (l *logger) Warning(format string, v ...interface{}) {
	if l.currentLevel <= WARNING {
		l.warnLogger.Printf(l.formatMessage(format), v...)
	}
}

func (l *logger) Error(format string, v ...interface{}) {
	if l.currentLevel <= ERROR {
		l.errorLogger.Printf(l.formatMessage(format), v...)
	}
}

func (l *logger) DebugStruct(name string, obj interface{}) {
	if l.currentLevel <= DEBUG {
		l.debugLogger.Printf(l.formatMessage("%s: %+v"), name, obj)
	}
}

func (l *logger) WithContext(ctx context.Context) Logger {
	newLogger := *l
	newLogger.fields = make(map[string]interface{})
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Aquí podrías extraer request ID u otros valores del contexto
	return &newLogger
}

func (l *logger) WithFields(fields map[string]interface{}) Logger {
	newLogger := *l
	newLogger.fields = make(map[string]interface{})
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return &newLogger
}

// Funciones globales para compatibilidad con código existente
func Debug(format string, v ...interface{}) {
	defaultLogger.Debug(format, v...)
}

func Info(format string, v ...interface{}) {
	defaultLogger.Info(format, v...)
}

func Warning(format string, v ...interface{}) {
	defaultLogger.Warning(format, v...)
}

func Error(format string, v ...interface{}) {
	defaultLogger.Error(format, v...)
}

func DebugStruct(name string, obj interface{}) {
	defaultLogger.DebugStruct(name, obj)
}

func WithFields(fields map[string]interface{}) Logger {
	return defaultLogger.WithFields(fields)
}
