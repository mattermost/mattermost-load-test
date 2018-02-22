FROM golang:1.10-alpine3.7
WORKDIR /go/src/github.com/mattermost/mattermost-load-test
COPY . .
RUN apk --no-cache add make
RUN make package

FROM alpine:3.7
RUN apk --no-cache add ca-certificates
WORKDIR /opt/mattermost-load-test
COPY --from=0 /go/src/github.com/mattermost/mattermost-load-test/dist/mattermost-load-test .

ENTRYPOINT ["./bin/loadtest"]
