package repo

import (
	"io"

	logger "github.com/sirupsen/logrus"
)

// NewLogger creates a logrus logger instance that writes to the given writer.
func NewLogger(w io.Writer) *logger.Logger {
	l := logger.New()
	l.SetOutput(w)
	l.SetFormatter(&logger.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
	})
	l.SetLevel(logger.DebugLevel)
	return l
}
