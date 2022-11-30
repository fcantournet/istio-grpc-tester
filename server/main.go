// Package main implements a server for Greeter service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	healthcheck "github.com/allisson/go-grpc-healthcheck"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	port      = ":50051"
	adminport = ":8888"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	mu       sync.Mutex
	slowdown time.Duration
	fail     bool
	failHC   bool
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.Name)
	if s.fail {
		return nil, fmt.Errorf("fail flag is true")
	}
	time.Sleep(s.slowdown)
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

// SayHelloHTTP is the http API
func (s *server) SayHelloHTTP(w http.ResponseWriter, r *http.Request) {
	if s.fail {
		w.WriteHeader(500)
	}
	time.Sleep(s.slowdown)
	w.WriteHeader(200)
}

func (s *server) SetSlowdown(w http.ResponseWriter, r *http.Request) {
	latencyString := r.URL.Query().Get("latency")
	latency, err := time.ParseDuration(latencyString)
	if err != nil {
		w.WriteHeader(503)
		return
	}
	s.mu.Lock()
	s.slowdown = latency
	s.mu.Unlock()
	w.WriteHeader(200)
	return
}

func (s *server) SetFail(w http.ResponseWriter, r *http.Request) {
	fail := r.URL.Query().Get("fail")

	s.mu.Lock()
	defer s.mu.Unlock()

	switch fail {
	case "true":
		s.fail = true
	case "false":
		s.fail = false
	default:
		w.WriteHeader(503)
		return
	}
	w.WriteHeader(200)
}

func (s *server) SetFailedHealthcheck(w http.ResponseWriter, r *http.Request) {
	fail := r.URL.Query().Get("fail")

	s.mu.Lock()
	defer s.mu.Unlock()

	switch fail {
	case "true":
		s.failHC = true
	case "false":
		s.failHC = false
	default:
		w.WriteHeader(503)
		return
	}
	w.WriteHeader(200)
}
func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	if s.failHC {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
	}
}

func (s *server) Check() error {
	if s.failHC {
		return errors.New("failing HC for grpc")
	}
	return nil
}

func main() {

	server := server{
		slowdown: time.Millisecond * 50,
		fail:     false,
	}

	mx := mux.NewRouter()
	mx.HandleFunc("/slowdown", server.SetSlowdown)
	mx.HandleFunc("/fail", server.SetFail)
	mx.HandleFunc("/healthcheck/fail", server.SetFailedHealthcheck)
	mx.HandleFunc("/ready", server.Ready)
	mx.HandleFunc("/hello", server.SayHelloHTTP)

	go http.ListenAndServe(adminport, mx)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	healthcheckServer := healthcheck.NewServer()
	healthcheckServer.AddChecker("failedHC-checker", &server)

	healthpb.RegisterHealthServer(s, &healthcheckServer)
	pb.RegisterGreeterServer(s, &server)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
