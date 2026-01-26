package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
	iampb "google.golang.org/genproto/googleapis/iam/v1" //nolint:staticcheck // Using standard genproto package
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/blackwell-systems/gcp-iam-emulator/internal/config"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/rest"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/server"
	"github.com/blackwell-systems/gcp-iam-emulator/internal/storage"
)

var (
	port              = flag.Int("port", 8080, "Port to listen on")
	httpPort          = flag.Int("http-port", 0, "HTTP REST port (0 = disabled)")
	configFile        = flag.String("config", "", "Path to policy config file (YAML)")
	watch             = flag.Bool("watch", false, "Watch config file for changes and hot reload")
	trace             = flag.Bool("trace", false, "Enable trace mode (log authz decisions)")
	explain           = flag.Bool("explain", false, "Enable verbose trace output (implies --trace)")
	traceOutput       = flag.String("trace-output", "", "Output file for JSON trace logs (implies --trace)")
	allowUnknownRoles = flag.Bool("allow-unknown-roles", false, "Enable wildcard role matching (compat mode, less strict)")
	version           = "0.4.0-dev"
)

func main() {
	flag.Parse()

	log.Printf("GCP IAM Emulator v%s", version)

	enableTrace := *trace || *explain || *traceOutput != ""
	
	iamServer := server.NewServer()
	iamServer.SetTrace(enableTrace)
	iamServer.SetAllowUnknownRoles(*allowUnknownRoles)
	
	if *explain {
		iamServer.SetExplain(true)
	}
	
	if *traceOutput != "" {
		if err := iamServer.SetTraceOutput(*traceOutput); err != nil {
			log.Fatalf("Failed to set trace output: %v", err)
		}
	}

	if *configFile != "" {
		if err := loadConfig(*configFile, iamServer); err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		if *watch {
			go watchConfig(*configFile, iamServer)
		}
	}

	if enableTrace {
		log.Printf("Trace mode: ENABLED (authz decisions will be logged)")
		if *explain {
			log.Printf("Explain mode: ENABLED (verbose trace output)")
		}
		if *traceOutput != "" {
			log.Printf("Trace output: %s (JSON format)", *traceOutput)
		}
	}
	
	if *allowUnknownRoles {
		log.Printf("Compat mode: ENABLED (wildcard role matching allowed - less strict)")
	} else {
		log.Printf("Strict mode: ENABLED (unknown roles denied - use --allow-unknown-roles for compat mode)")
	}

	if *httpPort > 0 {
		go startHTTPServer(*httpPort, iamServer.GetStorage(), *trace)
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

func startHTTPServer(port int, store *storage.Storage, trace bool) {
	restServer := rest.NewServer(store, trace)
	
	mux := http.NewServeMux()
	restServer.RegisterHandlers(mux)
	
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP REST server on port %d", port)
	
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	if err := httpServer.ListenAndServe(); err != nil {
		log.Printf("HTTP server error: %v", err)
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
	
	if len(cfg.Groups) > 0 {
		groups := make(map[string][]string)
		for groupName, groupCfg := range cfg.Groups {
			groups[groupName] = groupCfg.Members
		}
		iamServer.LoadGroups(groups)
		log.Printf("Loaded %d groups from config", len(groups))
	}
	
	if len(cfg.Roles) > 0 {
		roles := make(map[string][]string)
		for roleName, roleCfg := range cfg.Roles {
			roles[roleName] = roleCfg.Permissions
		}
		iamServer.LoadCustomRoles(roles)
		log.Printf("Loaded %d custom roles from config", len(roles))
	}
	
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
