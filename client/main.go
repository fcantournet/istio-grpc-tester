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
	"google.golang.org/grpc/metadata"
)

const (
	defaultName = "felix"
)

var address string
var port int
var period time.Duration
var sendhttp bool

func init() {
	flag.IntVar(&port, "port", 50051, "target grpc port")
	flag.StringVar(&address, "address", "localhost", "target grpc address")
	flag.BoolVar(&sendhttp, "http", false, "send http queries instead")
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

	var gracefulStop = make(chan os.Signal, 1)

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*600)
	defer cancel()

	md := metadata.New(map[string]string{"x-envoy-upstream-rq-timeout-ms": "60"})
	ctx = metadata.NewOutgoingContext(ctx, md)

	start := time.Now()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	duration := time.Since(start)
	if err != nil {
		log.Printf("Error: %v Duration: %v", err, duration)
		return
	}
	log.Printf("Success: %s Duration: %v", r.Message, duration)
}

func httpSayHelloForever(address string, port int, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	query := fmt.Sprintf("http://%v:%v/hello?name=%s", address, port, name)

	c := http.Client{
		Timeout: time.Millisecond * 500,
	}

	var gracefulStop = make(chan os.Signal, 1)

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
	// md := metadata.New(map[string]string{"x-envoy-upstream-rq-timeout-ms": "900"})
	// ctx = metadata.NewOutgoingContext(ctx, md)
	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		log.Fatalf("could not create request: %v", err)
	}
	req.Header.Add("x-envoy-upstream-rq-timeout-ms", "60")
	resp, err := c.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("HTTPError: %v Duration: %v", err, duration)
		return
	}
	defer resp.Body.Close()
	log.Printf("HTTPSuccess: %v Duration: %v", resp.StatusCode, duration)
}

func main() {
	flag.Parse()

	// Contact the server and print out its response.
	name := defaultName
	if len(flag.Args()) > 1 {
		name = flag.Arg(1)
	}

	var wg sync.WaitGroup

	if sendhttp {
		wg.Add(1)
		go httpSayHelloForever(address, port, name, &wg)
	} else {
		wg.Add(1)
		go grpcSayHelloForever(address, port, name, &wg)
	}
	wg.Wait()
}
