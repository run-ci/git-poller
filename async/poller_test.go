package async

import (
	"context"
	"testing"
)

type testPoller struct {
	ch chan struct{}
}

func (plr *testPoller) Poll(ctx context.Context) error {
	plr.ch <- struct{}{}

	return nil
}

func TestPoolAdd(t *testing.T) {
	errch := make(chan error)
	pollch := make(chan struct{})

	pool := NewPool()

	go func() {
		err := pool.Run()
		if err != nil {
			errch <- err
		}
	}()

	pool.AddPoller("test", &testPoller{ch: pollch})

	select {
	case <-pollch:
		t.Log("poller ran")
	case err := <-errch:
		t.Fatalf("expected no error, got %v", err)
	}
}
