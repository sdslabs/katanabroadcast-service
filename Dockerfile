FROM golang:1.17 as builder
WORKDIR /opt/katanabroadcast/
COPY . .
RUN go build -o ./katanabroadcast

FROM ubuntu:latest
WORKDIR /opt/katanabroadcast/
COPY --from=builder /opt/katanabroadcast/katanabroadcast ./
CMD ["./katanabroadcast"]