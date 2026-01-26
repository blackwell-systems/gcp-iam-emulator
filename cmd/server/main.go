package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/fsnotify/fsnotify"
	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/config"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/server"
)

var (
	port       = flag.Int("port", 8080, "Port to listen on")
	configFile = flag.String("config", "", "Path to policy config file (YAML)")
	watch      = flag.Bool("watch", false, "Watch config file for changes and hot reload")
	trace      = flag.Bool("trace", false, "Enable trace mode (log authz decisions)")
	version    = "0.2.0-dev"
)

func main() {
	flag.Parse()

	log.Printf("GCP IAM Emulator v%s", version)

	iamServer := server.NewServer()
	iamServer.SetTrace(*trace)

	if *configFile != "" {
		if err := loadConfig(*configFile, iamServer); err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		if *watch {
			go watchConfig(*configFile, iamServer)
		}
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
	iampb.RegisterIAMPolicyServer(grpcServer, iamServer) //nolint:staticcheck // Using standard genproto package
	reflection.Register(grpcServer)

	log.Printf("Server listening at %s", lis.Addr())
	log.Println("Ready to accept connections")

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serve: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig(path string, iamServer *server.Server) error {
	log.Printf("Loading policy config from %s", path)
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	policies := cfg.ToPolicies()
	iamServer.LoadPolicies(policies)
	log.Printf("Loaded %d policies from config", len(policies))
	return nil
}

func watchConfig(path string, iamServer *server.Server) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Failed to create file watcher: %v", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		log.Printf("Failed to watch config file: %v", err)
		return
	}

	log.Printf("Watching config file for changes: %s", path)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("Config file changed, reloading policies...")
				if err := loadConfig(path, iamServer); err != nil {
					log.Printf("Failed to reload config: %v", err)
				} else {
					log.Printf("Policies reloaded successfully")
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}
