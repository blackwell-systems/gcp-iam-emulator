package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/config"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/server"
)

var (
	port       = flag.Int("port", 8080, "Port to listen on")
	configFile = flag.String("config", "", "Path to policy config file (YAML)")
	trace      = flag.Bool("trace", false, "Enable trace mode (log authz decisions)")
	version    = "0.2.0-dev"
)

func main() {
	flag.Parse()

	log.Printf("GCP IAM Emulator v%s", version)

	iamServer := server.NewServer()
	iamServer.SetTrace(*trace)

	if *configFile != "" {
		log.Printf("Loading policy config from %s", *configFile)
		cfg, err := config.LoadFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		policies := cfg.ToPolicies()
		iamServer.LoadPolicies(policies)
		log.Printf("Loaded %d policies from config", len(policies))
	}

	if *trace {
		log.Printf("Trace mode: ENABLED (authz decisions will be logged)")
	}

	log.Printf("Starting gRPC server on port %d", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	iampb.RegisterIAMPolicyServer(grpcServer, iamServer)
	reflection.Register(grpcServer)

	log.Printf("Server listening at %s", lis.Addr())
	log.Println("Ready to accept connections")

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serve: %v\n", err)
		os.Exit(1)
	}
}
