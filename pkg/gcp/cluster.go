package gcp

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
)

type Client struct {
	clusterSvc *container.ClusterManagerClient
}

type GcloudError struct {
	msg string
}

func (g GcloudError) Error() string {
	return fmt.Sprintf("%s: did you run `gcloud auth application-default login`?", g.msg)
}

func newGcloudError(format string, args ...any) error {
	return GcloudError{fmt.Sprintf(format, args...)}
}

// NewClient will return a client that has permissions to interact with GCP services
func NewClient(clusterManager *container.ClusterManagerClient) (*Client, error) {
	return &Client{clusterSvc: clusterManager}, nil
}

// NewClusterClient creates a client that can be used to interact with GKE clusters
func NewClusterClient(ctx context.Context) (*container.ClusterManagerClient, error) {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, newGcloudError("create google cluster client: %s", err)
	}
	return c, nil
}

// GetCluster will return GKE cluster details
func (c *Client) GetCluster(ctx context.Context, cluster, location, project string) (*containerpb.Cluster, error) {
	query := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)
	req := &containerpb.GetClusterRequest{Name: query}

	resp, err := c.clusterSvc.GetCluster(ctx, req)
	if err != nil {
		return nil, newGcloudError("get GCP cluster %q: %s", query, err)
	}

	return resp, nil
}
