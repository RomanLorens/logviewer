package logger

import (
	l "github.com/RomanLorens/logger/log"
)

//L app logger
var L l.Logger

func init() {
	_logger, _ := l.New(l.WithConfig("logs/logviewer.log").Build())
	L = _logger
}
