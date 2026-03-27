ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.23

# --- Stage 1: Build stage ---

FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ssh-wrapper .

# --- Stage 2: Runtime stage ---
ARG ALPINE_VERSION=3.19
FROM alpine:${ALPINE_VERSION}

# Install required tools
RUN apk add --no-cache openssh-client git

# 1. Prepare original SSH
# Move real ssh to a location accessible only by root
RUN mv /usr/bin/ssh /usr/bin/ssh.orig && \
    chown 0:0 /usr/bin/ssh.orig && \
    chmod 700 /usr/bin/ssh.orig

# 2. Deploy our Go Wrapper
COPY --from=builder /app/ssh-wrapper /usr/bin/ssh

# Must be owned by root and set setuid (4755)
RUN chown 0:0 /usr/bin/ssh && \
    chmod 4755 /usr/bin/ssh

# 3. Create config directory (Secrets will be mounted here)
RUN mkdir -p /etc/ssh-wrapper && \
    chown 0:0 /etc/ssh-wrapper && \
    chmod 700 /etc/ssh-wrapper

# 4. Create low-privilege user
RUN adduser -D -u 1000 user
USER user
WORKDIR /home/user
