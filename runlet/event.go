package runlet

import (
	"github.com/run-ci/run/pkg/run"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Entry

func init() {
	logger = logrus.WithField("package", "event")
}

// Event represents a series of run tasks that can
// be run together as a pipeline. This is the format
// the runlet is expecting events in.
type Event struct {
	Name   string
	Remote string
	Branch string
	Steps  []Step
}

// Step is a logical grouping of tasks that can be run
// concurrently.
type Step struct {
	Name  string
	Tasks []run.Task
}

// Sender encapsulates the functionality of sending events
// to runlets or other interested parties. Implementations
// should track receipts and other things themselves. They
// should also be thread-safe.
type Sender interface {
	Send(Event) error
}
