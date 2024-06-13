package gcp

import (
	"context"
	"fmt"
	"net"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockClusterServer struct {
	containerpb.ClusterManagerServer
}

func setupMockClusterServer() (option.ClientOption, error) {
	srv := &mockClusterServer{}
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("create mock cluster server for test: %w", err)
	}

	gsrv := grpc.NewServer()
	containerpb.RegisterClusterManagerServer(gsrv, srv)
	go func() {
		if err := gsrv.Serve(l); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create gRPC client for test: %w", err)
	}

	return option.WithGRPCConn(conn), nil
}

func NewClusterMockClient() (*container.ClusterManagerClient, error) {
	// Should be fine to setup ephemeral server per client as only used for tests
	clientOpt, err := setupMockClusterServer()
	if err != nil {
		return nil, err
	}

	client, err := container.NewClusterManagerClient(context.Background(), clientOpt)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (f *mockClusterServer) GetCluster(ctx context.Context, req *containerpb.GetClusterRequest) (*containerpb.Cluster, error) {
	resp := &containerpb.Cluster{
		Name:      "gcp-predev-1234",
		Locations: []string{"us-west-2"},
	}
	return resp, nil
}
