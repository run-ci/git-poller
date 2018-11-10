package async

import "context"

// Poller encapsulates asynchronous polling behavior.
type Poller interface {
	Poll(ctx context.Context)
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
}

// NewPool returns a Pool with all its components initialized.
func NewPool() *Pool {
	return &Pool{
		// Do not access this directly. Only edit it by using
		// one of the control channels below.
		db: make(map[string]proc),

		addChan: make(chan msgAddProc),
	}
}

// Run runs the Pool's main loop. This listens for messages on
// its control channels to manipulate Pollers.
func (pool *Pool) Run() error {
	for {
		select {
		case addmsg := <-pool.addChan:
			logger.Debugf("adding: %+v", addmsg)

			ctx, kill := context.WithCancel(context.Background())
			ps := proc{
				kill: kill,
				plr:  addmsg.plr,
			}

			pool.db[addmsg.key] = ps
			go ps.plr.Poll(ctx)
		}
	}

	return nil
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
