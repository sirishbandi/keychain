# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY *.go .
COPY static/ ./static/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o server *.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine
WORKDIR /app
COPY --from=builder /workspace/server .
COPY static/ ./static/
COPY channel.jpg ./channel.jpg
RUN apt update && apt install -y imagemagick
USER 65532:65532