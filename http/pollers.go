package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/google/uuid"
	"github.com/run-ci/git-poller/runlet"
	"github.com/run-ci/run/pkg/run"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	yaml "gopkg.in/yaml.v2"
)

type pollerRequest struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

type pollerResponse pollerRequest

type gitPoller struct {
	remote   string
	branch   string
	agent    *run.Agent
	lastHead string

	queue chan<- []byte
}

func (gp *gitPoller) Poll(ctx context.Context) error {
	logger := logger.WithFields(logrus.Fields{
		"poll":   "git",
		"remote": gp.remote,
		"branch": gp.branch,
	})

	done := ctx.Done()
	for {
		select {
		case <-done:
			logger.Info("context done, shutting down poller")

			return nil
		default:
			logger.Info("running poller")

			err := gp.checkRepo()
			if err != nil {
				logger.WithError(err).Error("unable to clone git repo")
			}

			logger.Debug("sleeping")
			time.Sleep(1 * time.Minute)
		}
	}
}

func (gp *gitPoller) checkRepo() error {
	logger := logger.WithFields(logrus.Fields{
		"poll":   "git",
		"remote": gp.remote,
		"branch": gp.branch,
	})

	clonedir := fmt.Sprintf("/tmp/git-poller.%v", uuid.New())

	opts := &git.CloneOptions{
		URL:           gp.remote,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%v", gp.branch)),
		SingleBranch:  true,
		Depth:         1,
	}

	logger.Infof("cloning into %v", clonedir)

	repo, err := git.PlainClone(clonedir, false, opts)
	if err != nil {
		logger.WithError(err).Debug("unable to clone repo")
		return err
	}

	head, err := repo.Head()
	if err != nil {
		logger.WithError(err).Debug("unable to get repo HEAD")

		cleanerr := os.RemoveAll(clonedir)
		if err != nil {
			logger.WithError(cleanerr).Debugf("unable to clean up clonedir %v", clonedir)
		}

		return err
	}

	logger.Infof("got repo head %v", head)
	if head.String() != gp.lastHead {
		logger.Info("head changed, parsing pipelines")

		files, err := ioutil.ReadDir(fmt.Sprintf("%v/pipelines", clonedir))
		if err != nil {
			logger.WithError(err).Debug("unable to list pipeline files")
			cleanerr := os.RemoveAll(clonedir)
			if err != nil {
				logger.WithError(cleanerr).Debugf("unable to clean up clonedir %v", clonedir)
			}
			return err
		}
		if len(files) == 0 {
			cleanerr := os.RemoveAll(clonedir)
			if err != nil {
				logger.WithError(cleanerr).Debugf("unable to clean up clonedir %v", clonedir)
			}

			return nil
		}

		for _, finfo := range files {
			name := strings.Split(finfo.Name(), ".")[0]
			logger := logger.WithField("pipeline_name", name)

			path := fmt.Sprintf("%v/pipelines/%v", clonedir, finfo.Name())
			f, err := os.Open(path)
			if err != nil {
				logger.WithError(err).
					Debugf("unable to open pipeline file at %v, skipping", path)

				continue
			}

			buf, err := ioutil.ReadAll(f)
			if err != nil {
				logger.WithError(err).
					Debugf("unable to read pipeline file at %v, skipping", path)

				continue
			}

			var ev runlet.Event
			err = yaml.UnmarshalStrict(buf, &ev)
			if err != nil {
				logger.WithError(err).
					Debugf("unable to unmarshal pipeline for %v, skipping", path)

				continue
			}

			ev.Remote = gp.remote
			ev.Branch = gp.branch
			ev.Name = name

			logger = logger.WithField("event", ev)

			jsonbuf, err := json.Marshal(ev)
			if err != nil {
				logger.WithError(err).
					Debugf("unable to marshal event for %v, skipping", path)

				continue
			}

			gp.queue <- jsonbuf
		}

		gp.lastHead = head.String()
	}

	err = os.RemoveAll(clonedir)
	if err != nil {
		logger.WithError(err).Debugf("unable to clean up clonedir %v", clonedir)
		return err
	}

	logger.Debug("clonedir successfully deleted")
	return nil
}

func (srv *Server) runPoller(rw http.ResponseWriter, req *http.Request) {
	reqid := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqid)

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.WithField("error", err).Error("unable to read request body")

		rw.WriteHeader(http.StatusInternalServerError)
		buf, err := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if err != nil {
			return
		}
		rw.Write(buf)
		return
	}
	defer req.Body.Close()

	var body pollerRequest
	err = json.Unmarshal(buf, &body)
	if err != nil {
		logger.WithField("error", err).Error("unable to unmarshal request body")
		rw.WriteHeader(http.StatusBadRequest)

		buf, err := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if err != nil {
			return
		}
		rw.Write(buf)
		return
	}

	gp := &gitPoller{
		remote: body.Remote,
		branch: body.Branch,
		queue:  srv.queue,
	}

	srv.pool.AddPoller(fmt.Sprintf("%v#%v", gp.remote, gp.branch), gp)

	rw.WriteHeader(http.StatusAccepted)
}

func (srv *Server) removePoller(rw http.ResponseWriter, req *http.Request) {
	reqid := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqid)

	key, err := keyFromReq(req)
	if err != nil {
		logger.WithError(err).Error("unable to get poller key")

		rw.WriteHeader(http.StatusBadRequest)
		buf, err := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if err != nil {
			logger.WithField("marshal_err", err).
				Error("unable to marshal error response")

			return
		}
		rw.Write(buf)
		return
	}

	srv.pool.DeletePoller(key)

	rw.WriteHeader(http.StatusAccepted)
}

func (srv *Server) getPollers(rw http.ResponseWriter, req *http.Request) {
	reqid := req.Context().Value(keyReqID).(string)
	logger := logger.WithField("request_id", reqid)

	logger.Debug("begin getting pollers")
	plrs := srv.pool.GetPollers()
	logger.Debug("done getting pollers")

	resp := make([]pollerResponse, len(plrs))
	for i, plr := range plrs {
		tup := strings.Split(plr, "#")

		resp[i] = pollerResponse{
			Remote: tup[0],
			Branch: tup[1],
		}
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		logger.WithError(err).Error("unable to marshal response")

		rw.WriteHeader(http.StatusInternalServerError)
		buf, err = json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if err != nil {
			logger.WithField("marshal_err", err).
				Error("unable to marshal error response")

			return
		}
		rw.Write(buf)
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(buf)
}

func keyFromReq(req *http.Request) (string, error) {
	params := req.URL.Query()
	if val, ok := params["remote"]; !ok || len(val) == 0 {
		return "", errors.New("missing \"remote\" query parameter")
	}

	if val, ok := params["branch"]; !ok || len(val) == 0 {
		return "", errors.New("missing \"branch\" query parameter")
	}

	remote := params["remote"][0]
	branch := params["branch"][0]
	return fmt.Sprintf("%v#%v", remote, branch), nil
}
