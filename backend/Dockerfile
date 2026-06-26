FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./

ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,https://goproxy.io,direct
ENV GOSUMDB=off

RUN go mod download

COPY . .

RUN go build -o arke-server .

FROM alpine:3.19

WORKDIR /app

RUN mkdir -p /app/uploads /app/migrations /app/config

COPY --from=builder /app/arke-server .
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/config /app/config

ENV APP_ENV=prod

EXPOSE 8080

CMD ["./arke-server"]