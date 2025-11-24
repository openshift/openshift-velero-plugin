package test

import (
	"io"

	"github.com/sirupsen/logrus"
)

// NewLogger initialize test logger
func NewLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = io.Discard
	return logrus.NewEntry(logger)
}
