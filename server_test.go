package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/run-ci/git-poller/async"
)

type testpoller struct {
	ch chan struct{}
}

func (tp *testpoller) handleCreate(msg pollermsg) error {
	tp.ch <- struct{}{}

	return nil
}

func TestCreatePoller(t *testing.T) {
	pool := async.NewPool()
	go func() {
		err := pool.Run()
		if err != nil {
			logger.WithError(err).Fatal("unable to start poller pool, shutting down")
		}
	}()

	recv := make(chan []byte)
	send := make(chan struct{})

	srv := server{
		recv: recv,
		pool: pool,

		mux: make(map[string][]handlerFunc),
	}

	tp := testpoller{
		ch: send,
	}

	srv.handleFunc(msgOpCreate, tp.handleCreate)

	go func() {
		srv.run()
	}()

	testmsg := pollermsg{
		Op: msgOpCreate,
	}

	buf, err := json.Marshal(testmsg)
	if err != nil {
		t.Fatalf("got error marshalling testmsg: %v", err)
	}

	recv <- buf

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("expected create handler to be triggered immediately")
	case <-send:
	}
}
