# syntax-docker/dockerfile:1

FROM golang:1.24-alpine3.22 AS builder

WORKDIR /worker

COPY go.mod go.sum ./
RUN go mod download

COPY src ./src/
RUN go build -o worker ./src


FROM alpine:3.22 AS final
WORKDIR /worker
COPY --from=builder /worker/worker ./worker
CMD ["/worker/worker"]
