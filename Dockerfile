FROM golang:1.11

RUN go get github.com/cespare/reflex
ADD . /go/src/github.com/Scalingo/sand
WORKDIR /go/src/github.com/Scalingo/sand
EXPOSE 9999
RUN go install github.com/Scalingo/sand/cmd/...
CMD /go/bin/sand
