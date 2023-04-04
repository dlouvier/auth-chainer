FROM golang:1.20 AS builder

WORKDIR /app

COPY go/go.mod ./
COPY go/server.go ./

RUN go get -d -v ./...

RUN go mod download; go mod verify

EXPOSE 8080

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/auth-chainer

FROM scratch
COPY --from=builder /app/auth-chainer /app/auth-chainer

CMD ["/app/auth-chainer"]