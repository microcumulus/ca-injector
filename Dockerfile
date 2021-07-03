FROM golang:alpine

WORKDIR $GOPATH/src/app

ENTRYPOINT /app

ADD app /
