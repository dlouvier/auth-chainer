FROM golang:1.20

WORKDIR /app

COPY go/go.mod ./
COPY go/server.go ./

RUN go get -d -v ./...

RUN go mod download; go mod verify

EXPOSE 8080

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app

CMD ["/app/auth-chainer"]