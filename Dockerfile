FROM golang:alpine

ADD . /go/src/github.com/mattermost/mattermost-load-test

RUN go install github.com/mattermost/mattermost-load-test

ENTRYPOINT /go/bin/mattermost_load_test

CMD ["app"]
