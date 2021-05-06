FROM golang:1.13.8

WORKDIR /go/src/app

ADD src src

RUN go build src/katanabroadcast.go
CMD ["./katanabroadcast"]
