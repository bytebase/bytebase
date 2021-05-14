package bytebase

import (
	"fmt"
	"log"
)

type Level uint

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Logger struct {
}

func NewLogger() *Logger {
	logger := &Logger{}
	return logger
}

func (l *Logger) Print(level Level, v ...interface{}) {
	if level == FATAL {
		log.Fatalln(v...)
	} else {
		log.Println(v...)
	}
}

func (l *Logger) Printf(level Level, format string, v ...interface{}) {
	if level == FATAL {
		log.Fatalln(fmt.Sprintf(format, v...))
	} else {
		log.Println(fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Debug(v ...interface{}) {
	l.Print(DEBUG, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Printf(DEBUG, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.Print(INFO, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.Printf(INFO, format, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	l.Print(WARN, v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Printf(WARN, format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.Print(ERROR, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Printf(ERROR, format, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.Print(FATAL, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Printf(FATAL, format, v...)
}
