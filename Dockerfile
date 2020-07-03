FROM golang:1.14 as builder

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o server ./server/
RUN go build -o client ./client/

# Build a small image
FROM debian:buster

COPY --from=builder /build/client /
COPY --from=builder /build/server /

