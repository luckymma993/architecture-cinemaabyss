FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxy-service main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/proxy-service .
EXPOSE 8000
CMD ["./proxy-service"]
