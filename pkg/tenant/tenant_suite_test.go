package tenant

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestTenant(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tenant tests")
}
