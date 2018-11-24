package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/run-ci/git-poller/runlet"
	"github.com/run-ci/run/pkg/run"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	yaml "gopkg.in/yaml.v2"
)

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
