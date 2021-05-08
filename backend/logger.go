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

func (l *Logger) Log(level Level, v ...interface{}) {
	if level == FATAL {
		log.Fatalln(v...)
	} else {
		log.Println(v...)
	}
}

func (l *Logger) Logf(level Level, format string, v ...interface{}) {
	if level == FATAL {
		log.Fatalln(fmt.Sprintf(format, v...))
	} else {
		log.Println(fmt.Sprintf(format, v...))
	}
}
