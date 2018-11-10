package main

import (
	"os"

	nats "github.com/nats-io/go-nats"
	"github.com/run-ci/git-poller/async"
	"github.com/run-ci/git-poller/runlet"

	"github.com/run-ci/git-poller/http"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Entry
var natsURL string

func init() {
	lvl, err := logrus.ParseLevel(os.Getenv("POLLER_LOG_LEVEL"))
	if err != nil {
		lvl = logrus.InfoLevel
	}

	logrus.SetLevel(lvl)

	logger = logrus.WithField("package", "main")

	natsURL = os.Getenv("POLLER_NATS_URL")
	if natsURL == "" {
		logger.Infof("no nats url specified, defaulting to %v", nats.DefaultURL)
		natsURL = nats.DefaultURL
	}
}

func main() {
	logger.Info("booting server...")

	pool := async.NewPool()
	go func() {
		err := pool.Run()
		if err != nil {
			logger.WithError(err).Fatal("unable to start poller pool, shutting down")
		}
	}()

	queue, err := runlet.NewNatsSender(natsURL, "pipelines")
	if err != nil {
		logger.WithError(err).Fatal("unable to connect to NATS, shutting down")
	}

	srv := http.NewServer(":9002", pool, queue)

	if err := srv.ListenAndServe(); err != nil {
		logger.WithField("error", err).Fatal("shutting down server")
	}
}
