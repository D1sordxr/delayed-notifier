FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/worker ./cmd/worker/main.go

FROM scratch

WORKDIR /app

COPY --from=builder /app/api /app/api
COPY --from=builder /app/worker /app/worker

COPY --from=builder /app/configs ./configs
