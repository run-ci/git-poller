# git-poller

A server that polls git repos for changes.

## Quick Start

```
source env/local
run build
docker-compose up --build

# In another window
nats-sub pipelines

# In yet another window
source env/local
curl -XPOST -d "$TEST_POLLER" http://localhost:9002/pollers
```

## How It Works

Creating a poller will run a thread in the background that clones
a git repo every minute. If there's a change to the branch specified
when creating the poller, the repo's pipelines will be parsed and
each one will be queued up through NATS. Pollers can be deleted by
calling `DELETE /pollers?remote=${REMOTE_URL_ENCODED}&branch=${BRANCH}`.

```
# Make sure to put quotes around the URL, the "&" will screw everything up.
curl -XDELETE "http://localhost:9002/pollers?remote=${TEST_REMOTE_URL_ENCODE}&branch=master"
```
