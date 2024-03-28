package application

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestApplication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Application tests")
}
