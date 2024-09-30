package render

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/phuslu/log"
)

func TestApplication(t *testing.T) {
	log.DefaultLogger.SetLevel(log.PanicLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Template render tests")
}
