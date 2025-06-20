FROM golang:1.24.0-alpine AS builder-debug
WORKDIR /app
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -gcflags="all=-N -l" -o server ./cmd/main.go

FROM alpine AS debug
WORKDIR /app
COPY --from=builder-debug /app/server .
COPY --from=builder-debug /go/bin/dlv .
EXPOSE 8080 40000
CMD ["./dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./server"]

# Release build stage
FROM golang:1.24.0-alpine AS builder-release
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/main.go

FROM alpine AS release
ENV GIN_MODE=release
WORKDIR /app
COPY --from=builder-release /app/server .
EXPOSE 8080
CMD ["./server"]
