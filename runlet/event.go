package runlet

import (
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
	Name   string `json:"name"`
	Remote string `json:"remote"`
	Branch string `yaml:"branch" json:"branch"`
	Steps  []Step `yaml:"steps" json:"steps"`
}

// Step is a logical grouping of tasks that can be run
// concurrently.
type Step struct {
	Name  string `yaml:"name" json:"name"`
	Tasks []Task `yaml:"tasks" json:"tasks"`
}

// Task is a run task, but its arguments are actual
// arguments to use instead of metadata.
type Task struct {
	Name      string                 `yaml:"name" json:"name"`
	Arguments map[string]interface{} `yaml:"arguments" json:"arguments"`
}
