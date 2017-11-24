FROM golang:1.9.2

RUN go version
RUN go get -v -u github.com/Sirupsen/logrus
RUN go get -v -u github.com/mitchellh/go-homedir
RUN go get -v -u github.com/pkg/errors

ADD . src/github.com/adamlamar/rungo

# Build binary in $GOPATH/bin/
RUN go install github.com/adamlamar/rungo

# Stage any downloaded tarballs
RUN mkdir -p $HOME/.go
RUN cp -r $GOPATH/src/github.com/adamlamar/rungo/go-releases/* $HOME/.go/

# Verify the correct go versions are executed
RUN GO_VERSION=1.8 $GOPATH/bin/rungo version | grep "go version go1.8 linux/amd64"
RUN echo "1.9.1" > $GOPATH/.go-version && $GOPATH/bin/rungo version | grep "go version go1.9.1 linux/amd64" && rm $GOPATH/.go-version
# TODO: test system version
# RUN $GOPATH/bin/rungo version | grep "go version go1.9.2 linux/amd64"
