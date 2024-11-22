package localintegration

import (
	"testing"

	"github.com/coreeng/corectl/pkg/logger"
	_ "github.com/coreeng/corectl/tests/localintegration/update"
	_ "github.com/coreeng/corectl/tests/localintegration/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

// For any commands which do not need write access to github apis or local config to be initialised
func TestLocalSuite(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = zap.NewNop()
	defer func() { logger.Log = oldLogger }()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local integration tests")
}
