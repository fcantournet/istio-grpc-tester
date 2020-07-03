// Package main implements a client for Greeter service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	address     = "retry-tester-server:50051"
	defaultName = "felix"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBackoffConfig(grpc.DefaultBackoffConfig))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	var gracefulStop = make(chan os.Signal)

	signal.Notify(gracefulStop, syscall.SIGTERM)

	tickchan := time.Tick(time.Millisecond * 500)

	for {
		select {
		case <-gracefulStop:
			os.Exit(0)
		case <-tickchan:
			sayhello(c, name)
		}
	}
}

func sayhello(c pb.GreeterClient, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	duration := time.Since(start)
	if err != nil {
		log.Printf("Error: %v Duration: %v", err, duration)
		return
	}
	log.Printf("Success: %s Duration: %v", r.Message, duration)
}
