// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "felix"
)

var address string
var port int
var period time.Duration

func init() {
	flag.IntVar(&port, "port", 50051, "target grpc port")
	flag.StringVar(&address, "address", "localhost", "target grpc address")
	flag.DurationVar(&period, "period", time.Millisecond*500, "period for requests sent")
}

func grpcSayHelloForever(address string, port int, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	target := fmt.Sprintf("%v:%v", address, port)
	conn, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithBackoffConfig(grpc.DefaultBackoffConfig))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	var gracefulStop = make(chan os.Signal)

	signal.Notify(gracefulStop, syscall.SIGTERM)

	tickchan := time.Tick(period)

	for {
		select {
		case <-gracefulStop:
			return
		case <-tickchan:
			go grpcSayHello(c, name)
		}
	}
}

func grpcSayHello(c pb.GreeterClient, name string) {
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
	return
}

func httpSayHelloForever(address string, port int, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	query := fmt.Sprintf("%v:%v/hello?name=%s", address, port, name)

	c := http.Client{
		Timeout: time.Millisecond * 500,
	}

	var gracefulStop = make(chan os.Signal)

	signal.Notify(gracefulStop, syscall.SIGTERM)

	tickchan := time.Tick(period)

	for {
		select {
		case <-gracefulStop:
			return
		case <-tickchan:
			go httpSayHello(c, query)
		}
	}
}

func httpSayHello(c http.Client, query string) {
	start := time.Now()
	resp, err := c.Get(query)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Error: %v StatusCode: %v Duration: %v", err, resp.StatusCode, duration)
		return
	}
	defer resp.Body.Close()
	log.Printf("Success: %v Duration: %v", resp.StatusCode, duration)
	return
}

func main() {
	flag.Parse()

	// Contact the server and print out its response.
	name := defaultName
	if len(flag.Args()) > 1 {
		name = flag.Arg(1)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go grpcSayHelloForever(address, port, name, &wg)

	wg.Add(1)
	go httpSayHelloForever(address, port, name, &wg)

	wg.Wait()
}
