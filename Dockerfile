FROM golang:1.27 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY config.yaml ./config.yaml
RUN go mod download
COPY core/ ./core/
RUN CGO_ENABLED=0 GOOS=linux go build -o /qwrttqr-rate-limiter ./core/cmd
FROM alpine:3.20 AS runner
COPY --from=builder /qwrttqr-rate-limiter /
COPY --from=builder /app/config.yaml /config.yaml
EXPOSE 8080
CMD ["/qwrttqr-rate-limiter"]
