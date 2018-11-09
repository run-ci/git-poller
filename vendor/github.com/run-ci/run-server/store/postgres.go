package store

import (
	"database/sql"

	_ "github.com/lib/pq" // load the postgres driver
)

// Postgres is a RepoStore backed by PostgreSQL.
type Postgres struct {
	db *sql.DB
}

// NewPostgres returns a RepoStore backed by PostgreSQL. It connects to the
// database using connstr.
func NewPostgres(connstr string) (Repo, error) {
	logger = logger.WithField("store", "postgres")

	logger.Debug("connecting to database")

	db, err := sql.Open("postgres", connstr)
	if err != nil {
		logger.WithField("error", err).Debug("unable to connect to database")
		return nil, err
	}

	return &Postgres{
		db: db,
	}, nil
}

// CreateGitRepo saves the Git repository in Postgres.
func (pg *Postgres) CreateGitRepo(repo GitRepo) error {
	logger.Debugf("creating git repo for %v", repo.Remote)

	sqlinsert := `
	INSERT INTO git_repos (remote)
	VALUES
		($1);
	`

	_, err := pg.db.Exec(sqlinsert, repo.Remote)
	if err != nil {
		logger.WithField("error", err).
			Debugf("unable to create git repo for %v", repo.Remote)
	}
	return err
}

func (pg *Postgres) GetGitRepo(remote string) (GitRepo, error) {
	logger := logger.WithField("remote", remote)
	logger.Debug("getting git repo from postgres")

	sqlq := `
	SELECT * FROM git_repos
	WHERE remote = $1;
	`

	var repo GitRepo
	return repo, pg.db.QueryRow(sqlq, remote).Scan(&repo.Remote)
}

func (pg *Postgres) GetGitRepos() ([]GitRepo, error) {
	logger.Debug("getting git repos from postgres")

	sqlq := `
	SELECT remote FROM git_repos;
	`

	rows, err := pg.db.Query(sqlq)
	if err != nil {
		logger.WithField("error", err).Debug("unable to query database")
		return nil, err
	}

	repos := []GitRepo{}
	for rows.Next() {
		repo := GitRepo{}
		err := rows.Scan(&repo.Remote)
		if err != nil {
			logger.WithField("error", err).Debug("unable to scan row")
			return repos, err
		}
		repos = append(repos, repo)
	}

	return repos, nil
}
