FROM golang:1.23.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wallet-notifier ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/wallet-notifier /wallet-notifier
ENV APP_PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/wallet-notifier"]


