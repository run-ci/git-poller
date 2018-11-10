package main

import (
	"os"

	"github.com/run-ci/git-poller/async"

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

	pool := async.NewPool()
	go func() {
		err := pool.Run()
		if err != nil {
			logger.WithError(err).Panic("unable to start poller pool, panic")
		}
	}()

	srv := http.NewServer(":9002", pool)

	if err := srv.ListenAndServe(); err != nil {
		logger.WithField("error", err).Fatal("shutting down server")
	}
}
