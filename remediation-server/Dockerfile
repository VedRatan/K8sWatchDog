FROM golang:1.24-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o remediation-server .

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/remediation-server .

ENTRYPOINT ["./remediation-server"]