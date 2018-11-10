package async

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Poller encapsulates asynchronous polling behavior.
type Poller interface {
	Poll(ctx context.Context) error
}

type proc struct {
	kill context.CancelFunc
	plr  Poller
}

type msgAddProc struct {
	key string
	plr Poller
}

// Pool is a group of running Pollers.
type Pool struct {
	db map[string]proc

	addChan chan msgAddProc
	rmChan  chan string
}

// NewPool returns a Pool with all its components initialized.
func NewPool() *Pool {
	return &Pool{
		// Do not access this directly. Only edit it by using
		// one of the control channels below.
		db: make(map[string]proc),

		addChan: make(chan msgAddProc),
		rmChan:  make(chan string),
	}
}

// Run runs the Pool's main loop. This listens for messages on
// its control channels to manipulate Pollers.
func (pool *Pool) Run() error {
	for {
		select {
		case addmsg := <-pool.addChan:
			logger := logger.WithField("key", addmsg.key)
			logger.Debugf("adding: %+v", addmsg)

			ctx, kill := context.WithCancel(context.Background())
			ps := proc{
				kill: kill,
				plr:  addmsg.plr,
			}

			pool.db[addmsg.key] = ps
			go func(key string, logger *logrus.Entry) {
				logger.Debug("running poller")

				// Poll should block here while the poller runs.
				err := ps.plr.Poll(ctx)
				if err != nil {
					logger.WithError(err).Error("got error from returning poller")

					// Going to the map directly is no longer safe because we're running
					// in another goroutine here. If this poller exited with some kind of
					// error then it needs to be removed from the pool.
					pool.DeletePoller(key)
				}

				logger.Debug("poller returned with no error")
			}(addmsg.key, logger)

		case rmmsg := <-pool.rmChan:
			logger := logger.WithField("key", rmmsg)
			logger.Debug("removing poller")

			ps := pool.db[rmmsg]
			logger.Debug("killing poller")
			ps.kill()
		}
	}
}

// AddPoller adds a new Poller to the pool. If a Poller with the
// given key is already present, nothing will happen. To add a
// new version of the Poller with that key, call RemovePoller for
// that key first. This is to make it harder for consumers to
// shoot themselves in the foot.
func (pool *Pool) AddPoller(key string, plr Poller) {
	msg := msgAddProc{
		key: key,
		plr: plr,
	}

	pool.addChan <- msg
}

// DeletePoller deletes the Poller with the given key from the
// Pool. If no poller with the given key is present, nothing
// will happen.
func (pool *Pool) DeletePoller(key string) {
	pool.rmChan <- key
}
