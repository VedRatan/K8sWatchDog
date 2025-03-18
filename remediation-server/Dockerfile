FROM golang:1.24-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

# Required to embed build info into binary.
COPY .git /.git

WORKDIR /app

COPY go.mod go.sum ./

COPY logger/ logger/

ARG AGENT_DIR=remediation-server
WORKDIR /app/$AGENT_DIR

# Copy Go modules and manifests
COPY $AGENT_DIR/go.mod go.mod
COPY $AGENT_DIR/go.sum go.sum


RUN go mod download

COPY $AGENT_DIR/handlers handlers
COPY $AGENT_DIR/ai ai
COPY $AGENT_DIR/k8s k8s
COPY $AGENT_DIR/k8scontroller k8scontroller
COPY $AGENT_DIR/types types
COPY $AGENT_DIR/main.go main.go
COPY $AGENT_DIR/Makefile Makefile

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o remediation-server .

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/remediation-server/remediation-server .

ENTRYPOINT ["./remediation-server"]