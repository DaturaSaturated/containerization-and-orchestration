FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o batch main.go

FROM alpine:3.18
COPY --from=builder /app/batch /batch
CMD ["/batch"]