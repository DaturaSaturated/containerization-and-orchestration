FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o lab2-api .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/lab2-api .
EXPOSE 8080
CMD ["./lab2-api"]
