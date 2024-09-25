package tenant

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
)

func TestTenant(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tenant tests")
}
