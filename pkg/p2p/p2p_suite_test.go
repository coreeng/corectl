package p2p

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestP2P(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "P2P tests")
}
