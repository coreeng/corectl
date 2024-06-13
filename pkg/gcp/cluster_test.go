package gcp

import (
	"context"
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	gcptest "github.com/coreeng/corectl/pkg/testutil/gcp"
	"github.com/stretchr/testify/assert"
)

func TestGetCluster(t *testing.T) {
	c, err := gcptest.NewClusterMockClient()
	assert.NoError(t, err)

	req := &containerpb.GetClusterRequest{}
	resp, err := c.GetCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.Name, "gcp-predev-1234")
	assert.Equal(t, resp.Locations, []string{"us-west-2"})
}
