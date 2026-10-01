FROM golang:1.27-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/worker ./cmd/worker

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/api /app/api
COPY --from=builder /app/worker /app/worker
COPY --from=builder /app/configs ./configs

USER nobody
