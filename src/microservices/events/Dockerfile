FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o events-service main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/events-service .
EXPOSE 8082
CMD ["./events-service"]
