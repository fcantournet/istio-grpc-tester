// Package main implements a client for Greeter service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"flag"
	"fmt"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "felix"
)

var address string
var port int

func init() {
	flag.IntVar(&port, "port", 50051, "target grpc port")
	flag.StringVar(&address, "address", "localhost", "target grpc address")
}


func main() {
	flag.Parse()
	// Set up a connection to the server.
	target := fmt.Sprintf("%v:%v", address, port)
	conn, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithBackoffConfig(grpc.DefaultBackoffConfig))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(flag.Args()) > 1 {
		name = flag.Arg(1)
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
