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
nats-pub pollers "$TEST_CREATE_POLLER"
```

## How It Works

Creating a poller will run a thread in the background that clones
a git repo every minute. If there's a change to the branch specified
when creating the poller, the repo's pipelines will be parsed and
each one will be queued up through NATS. Pollers can be deleted in the
same way as they are created, with the "op" set to "delete".

```
nats-pub pollers "$TEST_DELETE_POLLER"
```
