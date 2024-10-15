FROM golang:1.22

RUN go install github.com/cespare/reflex@latest
ADD . /go/src/github.com/Scalingo/sand
WORKDIR /go/src/github.com/Scalingo/sand
EXPOSE 9999
ENV CGO_ENABLED=0
RUN go install -buildvcs=false github.com/Scalingo/sand/cmd/...
CMD /go/bin/sand
