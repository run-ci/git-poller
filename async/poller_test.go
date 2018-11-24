package async

import (
	"context"
	"testing"
)

type testPoller struct {
	ch chan struct{}

	pollfn func(context.Context) error
}

func (tp *testPoller) Poll(ctx context.Context) error {
	return tp.pollfn(ctx)
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

	plr := &testPoller{ch: pollch}

	plr.pollfn = func(ctx context.Context) error {
		plr.ch <- struct{}{}

		return nil
	}

	pool.AddPoller("test", plr)

	select {
	case <-pollch:
		t.Log("poller ran")
	case err := <-errch:
		t.Fatalf("expected no error, got %v", err)
	}
}
