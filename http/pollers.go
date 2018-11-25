package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

type pollerResponse struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
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
