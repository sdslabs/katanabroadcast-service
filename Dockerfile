FROM golang:1.17 as builder
WORKDIR /opt/katanad/
COPY . .
RUN go build -o ./katanad

FROM ubuntu:latest
WORKDIR /opt/katanad/
COPY --from=builder /opt/katanad/katanad ./
CMD ["./katanad"]