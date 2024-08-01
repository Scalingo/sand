FROM golang:1.22

RUN go install github.com/cespare/reflex@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest
ADD . /go/src/github.com/Scalingo/sand
WORKDIR /go/src/github.com/Scalingo/sand
EXPOSE 9999
RUN go install -buildvcs=false github.com/Scalingo/sand/cmd/...
CMD /go/bin/sand
