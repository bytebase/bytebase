package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bytebase/bytebase/backend/api/v1"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/service"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
)

var (
	grpcPort = flag.Int("grpc-port", 50051, "gRPC server port")
	httpPort = flag.Int("http-port", 8080, "HTTP server port")
)

func main() {
	flag.Parse()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived shutdown signal")
		cancel()
	}()

	// Initialize services (in a real scenario, you would inject dependencies here)
	// For this example, we'll create mock services
	service := initServices(ctx)

	// Start gRPC server
	go func() {
		grpcAddr := fmt.Sprintf(":%d", *grpcPort)
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		s := grpc.NewServer()
		v1pb.RegisterSensitiveApprovalServiceServer(s, service)

		fmt.Printf("gRPC server listening at %v\n", lis.Addr())
		if err := s.Serve(lis); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		httpAddr := fmt.Sprintf(":%d", *httpPort)
		router := mux.NewRouter()

		// Register HTTP handlers (in a real scenario, you would setup proper API routes)
		router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		})

		fmt.Printf("HTTP server listening at %v\n", httpAddr)
		srv := &http.Server{
			Addr:    httpAddr,
			Handler: router,
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	// Wait for shutdown
	<-ctx.Done()
	fmt.Println("Shutting down servers...")
}

// initServices initializes all required services.
func initServices(ctx context.Context) *v1.SensitiveApprovalService {
	// In a real scenario, you would:
	// 1. Connect to the database
	// 2. Create the store
	// 3. Initialize all required services

	// For this example, we'll create a mock implementation
	return &v1.SensitiveApprovalService{
		// Mock implementation
	}
}
