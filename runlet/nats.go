package runlet

import (
	"encoding/json"
	"math"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
)

type natsSender chan<- Event

// Send implements the Sender interface for the natsSender channel.
func (send natsSender) Send(ev Event) error {
	send <- ev
	return nil
}

// NewNatsSender returns a Sender backed by the NATS
// instance at the given url. Calls to Send will publish
// messages to the given subject.
func NewNatsSender(url, subject string) (Sender, error) {
	logger := logger.WithFields(logrus.Fields{
		"subject": subject,
	})

	logger.Info("connecting to nats")

	nc, err := nats.Connect(url)
	if err != nil {
		for i := 1; i <= 3; i++ {
			timeout := time.Duration(math.Pow(2, float64(i))) * time.Second

			logger.WithFields(logrus.Fields{
				"error": err,
			}).Warnf("error connecting to nats, retrying after %v seconds", timeout)

			time.Sleep(timeout)
			nc, err = nats.Connect(url)
			if err == nil {
				break
			}
		}
	}

	logger.Info("nats connection successful")

	send := make(chan Event)
	go func(logger *logrus.Entry, send <-chan Event, nc *nats.Conn, subject string) {
		for ev := range send {
			logger.Debugf("sending event: %+v", ev)

			buf, err := json.Marshal(ev)
			if err != nil {
				logger.WithError(err).WithField("event", ev).
					Error("unable to marshal event, skipping")

				continue
			}

			err = nc.Publish(subject, buf)
			if err != nil {
				logger.WithError(err).WithField("event", ev).
					Error("unable to send event")
			}
		}
	}(logger, send, nc, subject)

	return natsSender(send), nil
}
