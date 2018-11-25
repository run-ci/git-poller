package queue

import (
	"math"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
)

const natsQueueName = "poller"

// NATS encapsulates a connection to NATS, with functionality
// for creating channels to send and receive.
type NATS struct {
	conn *nats.Conn
}

// NewNATS establishes a connection to NATS.
func NewNATS(url string) (NATS, error) {
	conn, err := getNatsConn(url)
	if err != nil {
		return NATS{}, err
	}

	return NATS{
		conn: conn,
	}, nil
}

func getNatsConn(url string) (*nats.Conn, error) {
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

	return nc, err
}

// Close shuts down the underlying NATS connection.
func (q *NATS) Close() {
	q.conn.Close()
}

// SenderOn returns a channel to send messages on the given subject.
func (q *NATS) SenderOn(subj string) chan<- []byte {
	logger := logger.WithField("subject", subj)

	logger.Debug("setting up queue sender")

	send := make(chan []byte)
	go func(logger *logrus.Entry, send <-chan []byte, nc *nats.Conn, subj string) {
		for msg := range send {
			logger.Debugf("sending data: %s", msg)

			err := nc.Publish(subj, msg)
			if err != nil {
				logger.WithError(err).WithField("message", msg).
					Error("unable to send message")
			}
		}
	}(logger, send, q.conn, subj)

	logger.Debug("queue sender initialized successfully")

	return send
}

// ListenerOn returns a channel to receive messages on the given subject.
// If there's an error setting up the subscription, it's returned.
func (q *NATS) ListenerOn(subj string) (<-chan []byte, error) {
	logger := logger.WithField("subject", subj)
	recv := make(chan []byte)

	logger.Debug("setting up queue subscription")

	// The subscription can be useful for things like telemetry but for now
	// it's not gonna be used.
	_, err := q.conn.QueueSubscribe(subj, natsQueueName, func(msg *nats.Msg) {
		recv <- msg.Data
	})
	if err != nil {
		logger.WithError(err).Debugf("unable to subscribe for queue %v", natsQueueName)
		return nil, err
	}

	logger.Debug("queue subscription initialized successfully")

	return recv, nil
}
