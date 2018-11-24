package main

import (
	"os"

	nats "github.com/nats-io/go-nats"
	"github.com/run-ci/git-poller/async"
	"github.com/run-ci/git-poller/http"
	"github.com/run-ci/git-poller/queue"

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

	logger.Info("creating async pool")

	pool := async.NewPool()
	go func() {
		err := pool.Run()
		if err != nil {
			logger.WithError(err).Fatal("unable to start poller pool, shutting down")
		}
	}()

	logger.Info("creating NATS bus")

	bus, err := queue.NewNATS(natsURL)
	if err != nil {
		logger.WithError(err).Fatal("unable to connect to NATS, shutting down")
	}

	logrus.RegisterExitHandler(func() {
		bus.Close()
	})

	logger.Info("creating send queue for pipelines")
	send := bus.SenderOn("pipelines")

	logger.Info("creating listen queue for pollers")
	recv, err := bus.ListenerOn("pollers")
	if err != nil {
		logger.WithError(err).Fatal("unable to set up pollers subscritpion, shutting down")
	}

	go func() {
		httpsrv := http.NewServer(":9002", pool)
		if err := httpsrv.ListenAndServe(); err != nil {
			logger.WithError(err).Fatal("got error from HTTP server")
		}
	}()

	logger.Info("initializing and running server")
	srv := server{
		send: send,
		recv: recv,
		pool: pool,
	}

	srv.run()
}
