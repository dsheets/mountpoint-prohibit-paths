FROM golang:1.8-alpine

RUN apk add --no-cache git gcc musl-dev

WORKDIR /go/src/github.com/dsheets/mountpoint-prohibit-paths
RUN go get -d -v \
    github.com/sirupsen/logrus \
    github.com/pkg/errors \
    github.com/docker/go-plugins-helpers/sdk \
    gopkg.in/dsheets/go-plugins-helpers.v999/mountpoint \
    gopkg.in/dsheets/docker.v999/volume/mountpoint
COPY . .
RUN CC=gcc go build --ldflags '-linkmode external -extldflags "-static"' \
    -o docker-mountpoint-prohibit-paths .

FROM alpine:3.5

RUN mkdir -p /run/docker/plugins/prohibit-paths
COPY --from=0 /go/src/github.com/dsheets/mountpoint-prohibit-paths/docker-mountpoint-prohibit-paths docker-mountpoint-prohibit-paths

