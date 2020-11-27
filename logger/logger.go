package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	l "gopkg.in/natefinch/lumberjack.v2"
)

type contextKey string

const (
	//ReqID request id
	ReqID contextKey = "reqID"
	//UserKey user key
	UserKey contextKey = "user"
)

func init() {
	logPath := logPath()
	Info(context.Background(), "init logger with logPath %v", logPath)
	lumber := &l.Logger{
		Filename:   logPath,
		MaxSize:    2, // megabytes
		MaxBackups: 3,
		MaxAge:     7, // days
	}
	mw := io.MultiWriter(os.Stdout, lumber)
	log.SetOutput(mw)
}

func logPath() string {
	return "logs/logviewer.log"
}

//Info info
func Info(ctx context.Context, format string, args ...interface{}) {
	_log(ctx, "INFO", format, args...)
}

//Error error
func Error(ctx context.Context, format string, args ...interface{}) {
	_log(ctx, "ERROR", format, args...)
}

func _log(ctx context.Context, level string, msg string, args ...interface{}) {
	var user, req string
	user, _ = ctx.Value(UserKey).(string)
	req, _ = ctx.Value(ReqID).(string)
	formatter := "|%v|%v|%v|%v"
	m := fmt.Sprintf(msg, args...)
	log.Printf(formatter, strings.ToLower(user), req, level, m)
}

//Panicf panics
func Panicf(ctx context.Context, format string, arg ...interface{}) {
	Error(ctx, format, arg...)
	panic(fmt.Sprintf(format, arg...))
}
