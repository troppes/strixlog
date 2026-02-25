FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o app ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /build/app .
EXPOSE 8081
ENTRYPOINT ["./app"]
