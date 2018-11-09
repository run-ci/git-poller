package main

import (
	"os"

	"github.com/run-ci/git-poller/http"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Entry

func init() {
	lvl, err := logrus.ParseLevel(os.Getenv("POLLER_LOG_LEVEL"))
	if err != nil {
		lvl = logrus.InfoLevel
	}

	logrus.SetLevel(lvl)

	logger = logrus.WithField("package", "main")
}

func main() {
	logger.Info("booting server...")

	srv := http.NewServer(":9002")

	if err := srv.ListenAndServe(); err != nil {
		logger.WithField("error", err).Fatal("shutting down server")
	}
}
