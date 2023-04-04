FROM golang:1.20


WORKDIR /app
# Copy everything from the current directory to the PWD (Present Working Directory) inside the container
COPY go/go.mod ./
COPY go/*.go ./

# Download all the dependencies
RUN go get -d -v ./...

# Install the package
RUN go mod download; go mod verify


# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the executable

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app

CMD ["/app/auth-chainer"]