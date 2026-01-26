package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/server"
)

var (
	port    = flag.Int("port", 8080, "Port to listen on")
	version = "0.1.0"
)

func main() {
	flag.Parse()

	log.Printf("GCP IAM Emulator v%s", version)
	log.Printf("Starting gRPC server on port %d", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	iamServer := server.NewServer()
	iampb.RegisterIAMPolicyServer(grpcServer, iamServer)
	reflection.Register(grpcServer)

	log.Printf("Server listening at %s", lis.Addr())
	log.Println("Ready to accept connections")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
