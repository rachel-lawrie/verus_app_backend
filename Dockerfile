# --- Runtime stage
FROM alpine:latest as runtime
WORKDIR /root

# Install libc compatibility for Go binaries and bash
RUN apk add --no-cache libc6-compat bash go


# --- Build stage
FROM golang:1.23.4 AS builder
WORKDIR /app
ADD . /app

# Set environment variable to specify which environment to build for
ARG ENV=dev
ENV ENV=${ENV}


# Debug: Print Go version and environment details
RUN go version && go env


RUN go mod tidy


# Install CompileDaemon for hot reloading
RUN go install -mod=mod github.com/githubnemo/CompileDaemon@latest

# Build the binary for the specified environment and output it to /app/main
# TODO: review it for production environment
ENTRYPOINT CompileDaemon --build="go build ./cmd/$ENV/main.go" --command=./main
