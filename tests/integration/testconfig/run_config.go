package testconfig

import "sync"

var (
	TestRunId     string
	testRunIdOnce sync.Once
)

func SetTestRunId(id string) {
	testRunIdOnce.Do(func() {
		TestRunId = id
	})
}

func GetTestRunId() string {
	return TestRunId
}
