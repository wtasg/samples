# Build stage
FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app.exe main.go

# Run stage
FROM alpine:3.20.1
WORKDIR /app
COPY --from=builder /app/app.exe .
# Ensure log directory is present and writeable
RUN mkdir -p /var/log/app && chmod 777 /var/log/app
EXPOSE 8080
CMD ["./app.exe"]
