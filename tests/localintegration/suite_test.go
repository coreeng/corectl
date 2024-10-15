package localintegration

import (
	"testing"

	_ "github.com/coreeng/corectl/tests/localintegration/update"
	_ "github.com/coreeng/corectl/tests/localintegration/version"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/phuslu/log"
)

// For any commands which do not need write access to github apis or local config to be initialised
func TestLocalSuite(t *testing.T) {
	log.DefaultLogger.SetLevel(log.PanicLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local integration tests")
}
