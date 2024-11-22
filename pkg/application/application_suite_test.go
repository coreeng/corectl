package application

import (
	"testing"

	"github.com/coreeng/corectl/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func TestApplication(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = zap.NewNop()
	defer func() { logger.Log = oldLogger }()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Application tests")
}
