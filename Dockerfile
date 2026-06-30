FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o webcli .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -h /home/webcli webcli

COPY --from=builder /build/webcli /usr/local/bin/webcli

USER webcli
WORKDIR /home/webcli

EXPOSE 8931

ENTRYPOINT ["webcli"]
CMD ["serve", "--transport", "sse"]
