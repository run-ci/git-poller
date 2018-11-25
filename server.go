package main

import (
	"encoding/json"
	"fmt"

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
	send chan<- []byte
	pool *async.Pool

	mux map[string][]handlerFunc
}

func (s *server) handleFunc(op string, fn handlerFunc) {
	logger := logger.WithField("op", op)
	logger.Debug("registering handler")

	if _, ok := s.mux[op]; !ok {
		logger.Debug("first handler for this op")
		s.mux[op] = []handlerFunc{fn}
		return
	}

	s.mux[op] = append(s.mux[op], fn)
}

// TODO: clean shutdown
func (s *server) run() {
	// This duplicates the logger so we don't have to worry about sharing it later.
	logger := logger.WithFields(logrus.Fields{})
	logger.Debug("running server")

	for raw := range s.recv {
		logger.Debug("received message")

		var msg pollermsg
		err := json.Unmarshal(raw, &msg)
		if err != nil {
			logger.WithField("error", err).Error("unable to unmarshal message, skipping")
			continue
		}

		switch msg.Op {
		case msgOpCreate:
			for _, fn := range s.mux[msgOpCreate] {
				err := fn(msg)
				if err != nil {
					logger.WithError(err).
						Errorf("got error running a handler for %v", msgOpCreate)
				}
			}
			// logger := logger.WithFields(logrus.Fields{
			// 	"remote": msg.Remote,
			// 	"branch": msg.Branch,
			// })
			// logger.Info("got create request")

			// gp := &gitPoller{
			// 	remote: msg.Remote,
			// 	branch: msg.Branch,
			// 	queue:  s.send,
			// }

			// s.pool.AddPoller(fmt.Sprintf("%v#%v", gp.remote, gp.branch), gp)

		case msgOpDelete:
			logger := logger.WithFields(logrus.Fields{
				"remote": msg.Remote,
				"branch": msg.Branch,
			})
			logger.Info("got delete request")

			key := fmt.Sprintf("%v#%v", msg.Remote, msg.Branch)
			s.pool.DeletePoller(key)
		default:
			logger := logger.WithFields(logrus.Fields{
				"remote": msg.Remote,
				"branch": msg.Branch,
				"op":     msg.Op,
			})
			logger.Warn("got a message that can't be handled yet, skipping")

			continue
		}
	}
}
