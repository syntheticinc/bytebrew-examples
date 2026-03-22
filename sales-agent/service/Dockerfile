FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o service .

FROM alpine:3.20

COPY --from=builder /build/service /usr/local/bin/service

EXPOSE 3000

CMD ["service"]
