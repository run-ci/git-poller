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

	// signal channels
	addChan chan msgAddProc
	rmChan  chan string
	getChan chan struct{}

	// This is for output of keys coming from the pool.
	outChan chan string
}

// NewPool returns a Pool with all its components initialized.
func NewPool() *Pool {
	return &Pool{
		// Do not access this directly. Only edit it by using
		// one of the control channels below.
		db: make(map[string]proc),

		addChan: make(chan msgAddProc),
		rmChan:  make(chan string),
		getChan: make(chan struct{}),

		outChan: make(chan string),
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

		case <-pool.getChan:
			// In order to keep everything safe without needing a lock, a "get" request
			// needs to be processed in this loop as well.
			for k := range pool.db {
				pool.outChan <- k
			}

			close(pool.outChan)
			pool.outChan = make(chan string)
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

// GetPollers returns a list of all poller keys. If keys are passed
// in, the pool is checked for a matching poller. If at least one
// of the keys isn't found, an error is returned.
func (pool *Pool) GetPollers() []string {
	keys := make([]string, 0, len(pool.db))

	pool.getChan <- struct{}{}

	for k := range pool.outChan {
		keys = append(keys, k)
	}

	return keys
}
