# Build stage
FROM golang:1.25-alpine AS builder
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server/

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/server .
EXPOSE 8787
ENTRYPOINT ["/app/server"]
