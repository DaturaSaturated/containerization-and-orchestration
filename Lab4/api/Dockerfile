FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

WORKDIR /app

COPY --from=builder --chown=1000:1000 /out/app /app/app

USER 1000:1000

EXPOSE 8080

ENTRYPOINT ["/app/app"]
