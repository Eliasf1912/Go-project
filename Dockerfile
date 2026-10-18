FROM golang:1.27.0-alpine3.24

RUN addgroup -g 1000 go && adduser -h /home/go -G go -D -u 1000 go

RUN mkdir -p /home/go/.cache /go/pkg/mod && chown -R go:go /home/go /go/pkg/mod
