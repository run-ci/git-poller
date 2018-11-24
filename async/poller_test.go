package async

import (
	"context"
	"testing"
	"time"
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

	if len(pool.db) != 1 {
		t.Fatalf("expected pool database to have one poller, got %v", len(pool.db))
	}

	// We know this works because if not the test will fail due to deadlock.
	select {
	case <-pollch:
		t.Log("poller ran")
	case err := <-errch:
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPoolRemove(t *testing.T) {
	errch := make(chan error)

	pool := NewPool()

	go func() {
		err := pool.Run()
		if err != nil {
			errch <- err
		}
	}()

	plr := &testPoller{
		ch: make(chan struct{}),
	}

	plr.pollfn = func(ctx context.Context) error {
		// This should just run until it's deleted.
		for {
			select {
			case <-ctx.Done():
				plr.ch <- struct{}{}
			}
		}
	}

	pool.AddPoller("test", plr)
	pool.DeletePoller("test")

	select {
	case <-time.After(1 * time.Second):
		// This should exit immediately if the context cancel func is called.
		t.Fatal("expected poller to be killed before 1 second")
	case <-plr.ch:
		if len(pool.db) != 0 {
			t.Fatalf("expected pool database to have no pollers, got %v", len(pool.db))
		}
	}
}
