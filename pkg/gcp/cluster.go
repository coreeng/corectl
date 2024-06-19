package gcp

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	googleContainer "google.golang.org/api/container/v1"
)

type Client struct {
	clusterSvc *container.ClusterManagerClient
}

// NewClient will return a client that has permissions to interact with GCP services
func NewClient(ctx context.Context, clusterManager *container.ClusterManagerClient) (*Client, error) {
	return &Client{clusterSvc: clusterManager}, nil
}

// NewContainerClient creates a client that can be used to fetch cluster credentials
func NewContainerClient(ctx context.Context) (*googleContainer.Service, error) {
	c, err := googleContainer.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("google create container client: %w", err)
	}
	return c, nil
}

// NewClusterClient creates a client that can be used to interact with GKE clusters
func NewClusterClient(ctx context.Context) (*container.ClusterManagerClient, error) {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create google cluster client: %w", err)
	}
	return c, nil
}

// GetCluster will return GKE cluster details
func (c *Client) GetCluster(ctx context.Context, cluster, location, project string) (*containerpb.Cluster, error) {
	query := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)
	req := &containerpb.GetClusterRequest{Name: query}

	resp, err := c.clusterSvc.GetCluster(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get GCP cluster: %w", err)
	}

	return resp, nil
}
