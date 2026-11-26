FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o blocking-target main.go

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/blocking-target .
EXPOSE 8090
ENTRYPOINT ["./blocking-target"]
