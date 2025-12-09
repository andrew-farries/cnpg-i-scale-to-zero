package main

import (
	"context"
	"log"
	"net"

	hibernationv1 "github.com/xataio/cnpg-i-scale-to-zero/gen/proto/hibernation/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	hibernationv1.UnimplementedHibernationServiceServer
}

func (s *server) HibernateCluster(ctx context.Context, req *hibernationv1.HibernateClusterRequest) (*hibernationv1.HibernateClusterResponse, error) {
	return &hibernationv1.HibernateClusterResponse{
		Success: true,
		Message: "no-op",
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	hibernationv1.RegisterHibernationServiceServer(s, &server{})
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
