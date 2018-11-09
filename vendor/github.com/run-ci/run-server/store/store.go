package store

import "github.com/sirupsen/logrus"

var logger *logrus.Entry

func init() {
	logger = logrus.WithField("package", "store")
}

// Repo is anything that can hold data about source repositories.
type Repo interface {
	CreateGitRepo(GitRepo) error
	GetGitRepo(string) (GitRepo, error)
	GetGitRepos() ([]GitRepo, error)
}

// GitRepo is a Git repository.
type GitRepo struct {
	Remote string `json:"remote"`
}
