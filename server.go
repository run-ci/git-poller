package main

import (
	"encoding/json"

	"github.com/run-ci/git-poller/async"
	"github.com/sirupsen/logrus"
)

const (
	msgOpCreate = "create"
	msgOpDelete = "delete"
)

type pollermsg struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
	Op     string `json:"op"`
}

type handlerFunc func(pollermsg) error

type server struct {
	recv <-chan []byte
	pool *async.Pool

	mux map[string]handlerFunc
}

func (s *server) handleFunc(op string, fn handlerFunc) {
	logger := logger.WithField("op", op)
	logger.Debug("registering handler")

	if _, ok := s.mux[op]; ok {
		logger.Warn("overriding previous handler")
	}

	s.mux[op] = fn
}

// TODO: clean shutdown
func (s *server) run() {
	for raw := range s.recv {
		logger.Debug("received message")

		var msg pollermsg
		err := json.Unmarshal(raw, &msg)
		if err != nil {
			logger.WithField("error", err).Error("unable to unmarshal message, skipping")
			continue
		}

		fn, ok := s.mux[msg.Op]
		if !ok {
			logger := logger.WithFields(logrus.Fields{
				"remote": msg.Remote,
				"branch": msg.Branch,
				"op":     msg.Op,
			})
			logger.Warn("got a message that can't be handled, skipping")

			continue
		}

		logger := logger.WithFields(logrus.Fields{
			"remote": msg.Remote,
			"branch": msg.Branch,
		})
		logger.Debugf("got %v request", msg.Op)

		err = fn(msg)
		if err != nil {
			logger.WithError(err).
				Errorf("got error running a handler for %v", msg.Op)
		}
	}
}
