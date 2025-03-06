
FROM golang:1.24-alpine AS builder

WORKDIR /app


COPY go.mod go.sum ./


RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -o k8s-agent .


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/k8s-agent .


CMD ["./k8s-agent"]