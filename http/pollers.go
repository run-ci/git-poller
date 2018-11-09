package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type pollerRequest struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
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

	go func(remote, branch string, logger *logrus.Entry) {
		for {
			logger.Info("running poller")

			logger.Debug("sleeping")
			time.Sleep(1 * time.Minute)
		}
	}(body.Remote, body.Branch, logger.WithFields(logrus.Fields{
		"remote": body.Remote,
		"branch": body.Branch,
	}))
}
