FROM ubuntu:18.04

RUN apt-get update && apt-get install -y ca-certificates

ADD git-poller /bin/git-poller

ENTRYPOINT [ "/bin/git-poller" ]
