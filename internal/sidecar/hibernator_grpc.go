package sidecar

import (
	"context"
	"fmt"

	pb "github.com/xataio/cnpg-i-scale-to-zero/gen/proto/hibernation/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCHibernator hibernates clusters by calling an external gRPC service. The
// service is responsible for handling hibernation.
type GRPCHibernator struct {
	client pb.HibernationServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCHibernator creates a new GRPCHibernator that connects to the given
// address.
func NewGRPCHibernator(addr string) (*GRPCHibernator, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to hibernation service: %w", err)
	}

	return &GRPCHibernator{
		client: pb.NewHibernationServiceClient(conn),
		conn:   conn,
	}, nil
}

// Hibernate triggers hibernation by calling the external gRPC service.
func (h *GRPCHibernator) Hibernate(ctx context.Context, namespace, name string) error {
	resp, err := h.client.HibernateCluster(ctx, &pb.HibernateClusterRequest{
		Namespace: namespace,
		Name:      name,
	})
	if err != nil {
		return fmt.Errorf("failed to call hibernation service: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("hibernation failed: %s", resp.Message)
	}

	return nil
}

// Close closes the gRPC connection.
func (h *GRPCHibernator) Close() error {
	if h.conn != nil {
		return h.conn.Close()
	}
	return nil
}
