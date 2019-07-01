package test

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

// NewLogger initialize test logger
func NewLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = ioutil.Discard
	return logrus.NewEntry(logger)
}
