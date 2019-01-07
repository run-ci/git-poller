package http

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/run-ci/git-poller/async"
)

type testPoller struct {
	ch chan struct{}

	pollfn func(context.Context) error
}

func (tp *testPoller) Poll(ctx context.Context) error {
	return tp.pollfn(ctx)
}

func TestGetPollers(t *testing.T) {
	req := httptest.NewRequest("GET", "http://test/pollers", nil)
	rw := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(context.Background(), keyReqID, "test"))

	pool := async.NewPool()
	go func() {
		// Don't care about errors from this here. This really
		// shouldn't error at all anyways.
		_ = pool.Run()
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

	pool.AddPoller("repo#master", plr)

	srv := NewServer("test:80", pool)
	srv.getPollers(rw, req)

	resp := rw.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %v, got %v", http.StatusOK, resp.StatusCode)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("got error reading response body: %v", err)
	}
	defer resp.Body.Close()

	body := []map[string]string{}
	err = json.Unmarshal(buf, &body)
	if err != nil {
		t.Fatalf("got error unmarshaling response body: %v", err)
	}

	if len(body) != 1 {
		t.Fatalf("expected body to have only one poller, got %v", len(body))
	}

	obj := body[0]
	if remote, ok := obj["remote"]; !ok || remote != "repo" {
		t.Fatalf(`expected "remote" to be set to "repo"; got %v`, remote)
	}

	if branch, ok := obj["branch"]; !ok || branch != "master" {
		t.Fatalf(`expected "branch" to be set to "master"; got %v`, branch)
	}
}
