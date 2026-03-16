package testconfig

import (
	"strings"
	"sync"
)

var (
	TestRunId     string
	testRunIdOnce sync.Once
)

func SetTestRunId(id string) {
	testRunIdOnce.Do(func() {
		TestRunId = strings.ToLower(id)
	})
}

func GetTestRunId() string {
	return TestRunId
}
